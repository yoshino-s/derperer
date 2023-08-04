package derperer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/tcnksm/go-httpstat"
	"go.uber.org/zap"
	"tailscale.com/derp/derphttp"
	"tailscale.com/net/netmon"
	"tailscale.com/net/netns"
	"tailscale.com/net/ping"
	"tailscale.com/tailcfg"
	"tailscale.com/types/logger"
	"tailscale.com/util/cmpx"
)

type probeProto uint8

const (
	probeIPv4  probeProto = iota // STUN IPv4
	probeIPv6                    // STUN IPv6
	probeHTTPS                   // HTTPS
)

type tester struct {
	ctx                 context.Context
	Logf                logger.Logf
	NetMon              *netmon.Monitor
	Pinger              *ping.Pinger
	overallProbeTimeout time.Duration
	icmpProbeTimeout    time.Duration
	latencyLimit        time.Duration
}

func newTester(ctx context.Context, logf logger.Logf, latencyLimit time.Duration, overallProbeTimeout time.Duration, icmpProbeTimeout time.Duration) (*tester, error) {
	netMon, err := netmon.New(logf)
	if err != nil {
		return nil, err
	}
	return &tester{
		ctx:                 ctx,
		Logf:                logf,
		Pinger:              ping.New(ctx, logf, netns.Listener(logf, netMon)),
		NetMon:              netMon,
		overallProbeTimeout: overallProbeTimeout,
		icmpProbeTimeout:    icmpProbeTimeout,
		latencyLimit:        latencyLimit,
	}, nil
}

func (t *tester) Test(derpMap *tailcfg.DERPMap) *tailcfg.DERPMap {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	newDerpMap := derpMap.Clone()

	zap.L().Debug("Start Test", zap.Int("regionCount", len(derpMap.Regions)))

	removeRegion := func(regionID int) {
		mu.Lock()
		defer mu.Unlock()
		delete(newDerpMap.Regions, regionID)
		zap.L().Debug("Remove Region", zap.Int("regionID", regionID))
	}

	for _, region := range derpMap.Regions {
		wg.Add(2)
		go func(region *tailcfg.DERPRegion) {
			defer wg.Done()
			latency, err := t.measureICMPLatency(t.ctx, region, t.Pinger)
			if err != nil {
				zap.L().Debug("ICMP Error", zap.Any("region", region.RegionName), zap.Error(err))
				removeRegion(region.RegionID)
			} else if latency > t.latencyLimit {
				zap.L().Debug("ICMP Latency", zap.Any("region", region.RegionName), zap.Duration("latency", latency))
				removeRegion(region.RegionID)
			}
		}(region)
		go func(region *tailcfg.DERPRegion) {
			defer wg.Done()
			latency, _, err := t.measureHTTPSLatency(t.ctx, region)
			if err != nil {
				zap.L().Debug("HTTPS Error", zap.Any("region", region.RegionName), zap.Error(err))
				removeRegion(region.RegionID)
			} else if latency > t.latencyLimit {
				zap.L().Debug("HTTPS Latency", zap.Any("region", region.RegionName), zap.Duration("latency", latency))
				removeRegion(region.RegionID)
			}
		}(region)
		wg.Wait()
		time.Sleep(1 * time.Second)
	}

	zap.L().Debug("End Test", zap.Int("regionCount", len(newDerpMap.Regions)))

	return newDerpMap
}

func (t *tester) measureHTTPSLatency(ctx context.Context, reg *tailcfg.DERPRegion) (time.Duration, netip.Addr, error) {
	var result httpstat.Result
	ctx, cancel := context.WithTimeout(httpstat.WithHTTPStat(ctx, &result), t.overallProbeTimeout)
	defer cancel()

	var ip netip.Addr

	dc := derphttp.NewNetcheckClient(t.Logf)
	defer dc.Close()

	tlsConn, tcpConn, node, err := dc.DialRegionTLS(ctx, reg)
	if err != nil {
		return 0, ip, err
	}
	defer tcpConn.Close()

	if ta, ok := tlsConn.RemoteAddr().(*net.TCPAddr); ok {
		ip, _ = netip.AddrFromSlice(ta.IP)
		ip = ip.Unmap()
	}
	if ip == (netip.Addr{}) {
		return 0, ip, fmt.Errorf("no unexpected RemoteAddr %#v", tlsConn.RemoteAddr())
	}

	connc := make(chan *tls.Conn, 1)
	connc <- tlsConn

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, errors.New("unexpected DialContext dial")
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			select {
			case nc := <-connc:
				return nc, nil
			default:
				return nil, errors.New("only one conn expected")
			}
		},
	}
	hc := &http.Client{Transport: tr}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://"+node.HostName+"/derp/latency-check", nil)
	if err != nil {
		return 0, ip, err
	}

	resp, err := hc.Do(req)
	if err != nil {
		return 0, ip, err
	}
	defer resp.Body.Close()

	// DERPs should give us a nominal status code, so anything else is probably
	// an access denied by a MITM proxy (or at the very least a signal not to
	// trust this latency check).
	if resp.StatusCode > 299 {
		return 0, ip, fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, resp.Status)
	}

	_, err = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	if err != nil {
		return 0, ip, err
	}
	result.End(time.Now())

	// TODO: decide best timing heuristic here.
	// Maybe the server should return the tcpinfo_rtt?
	return result.ServerProcessing, ip, nil
}

func (t *tester) measureICMPLatency(ctx context.Context, reg *tailcfg.DERPRegion, p *ping.Pinger) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, t.icmpProbeTimeout)
	defer cancel()

	if len(reg.Nodes) == 0 {
		return 0, fmt.Errorf("no nodes for region %d (%v)", reg.RegionID, reg.RegionCode)
	}

	// Try pinging the first node in the region
	node := reg.Nodes[0]

	// Get the IPAddr by asking for the UDP address that we would use for
	// STUN and then using that IP.
	//
	// TODO(andrew-d): this is a bit ugly
	nodeAddr := t.nodeAddr(ctx, node, probeIPv4)
	if !nodeAddr.IsValid() {
		return 0, fmt.Errorf("no address for node %v", node.Name)
	}
	addr := &net.IPAddr{
		IP:   net.IP(nodeAddr.Addr().AsSlice()),
		Zone: nodeAddr.Addr().Zone(),
	}

	// Use the unique node.Name field as the packet data to reduce the
	// likelihood that we get a mismatched echo response.
	return p.Send(ctx, addr, []byte(node.Name))
}

func (t *tester) nodeAddr(ctx context.Context, n *tailcfg.DERPNode, proto probeProto) (ap netip.AddrPort) {
	port := cmpx.Or(n.STUNPort, 3478)
	if port < 0 || port > 1<<16-1 {
		return
	}
	if n.STUNTestIP != "" {
		ip, err := netip.ParseAddr(n.STUNTestIP)
		if err != nil {
			return
		}
		if proto == probeIPv4 && ip.Is6() {
			return
		}
		if proto == probeIPv6 && ip.Is4() {
			return
		}
		return netip.AddrPortFrom(ip, uint16(port))
	}

	switch proto {
	case probeIPv4:
		if n.IPv4 != "" {
			ip, _ := netip.ParseAddr(n.IPv4)
			if !ip.Is4() {
				return
			}
			return netip.AddrPortFrom(ip, uint16(port))
		}
	case probeIPv6:
		if n.IPv6 != "" {
			ip, _ := netip.ParseAddr(n.IPv6)
			if !ip.Is6() {
				return
			}
			return netip.AddrPortFrom(ip, uint16(port))
		}
	default:
		return
	}

	// The default lookup function if we don't set UseDNSCache is to use net.DefaultResolver.
	lookupIPAddr := func(ctx context.Context, host string) ([]netip.Addr, error) {
		addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}

		var naddrs []netip.Addr
		for _, addr := range addrs {
			na, ok := netip.AddrFromSlice(addr.IP)
			if !ok {
				continue
			}
			naddrs = append(naddrs, na.Unmap())
		}
		return naddrs, nil
	}

	probeIsV4 := proto == probeIPv4
	addrs, _ := lookupIPAddr(ctx, n.HostName)
	for _, a := range addrs {
		if (a.Is4() && probeIsV4) || (a.Is6() && !probeIsV4) {
			return netip.AddrPortFrom(a, uint16(port))
		}
	}
	return
}
