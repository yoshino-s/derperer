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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tcnksm/go-httpstat"
	"go.uber.org/zap"
	"tailscale.com/derp/derphttp"
	"tailscale.com/net/netaddr"
	"tailscale.com/net/netmon"
	"tailscale.com/net/netns"
	"tailscale.com/net/ping"
	"tailscale.com/net/stun"
	"tailscale.com/tailcfg"
	"tailscale.com/types/logger"
	"tailscale.com/types/nettype"
)

type tester struct {
	ctx          context.Context
	Logf         logger.Logf
	NetMon       *netmon.Monitor
	Pinger       *ping.Pinger
	probeTimeout time.Duration
	latencyLimit time.Duration
}

func newTester(ctx context.Context, logf logger.Logf, latencyLimit time.Duration, probeTimeout time.Duration) (*tester, error) {
	netMon, err := netmon.New(logf)
	if err != nil {
		return nil, err
	}
	return &tester{
		ctx:          ctx,
		Logf:         logf,
		Pinger:       ping.New(ctx, logf, netns.Listener(logf, netMon)),
		NetMon:       netMon,
		probeTimeout: probeTimeout,
		latencyLimit: latencyLimit,
	}, nil
}

func flattenDEPRMap(derpMap *tailcfg.DERPMap) *tailcfg.DERPMap {
	res := &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}
	idx := 0
	for regionID, region := range derpMap.Regions {
		for _, node := range region.Nodes {
			res.Regions[idx] = &tailcfg.DERPRegion{
				RegionID:   idx,
				RegionName: fmt.Sprintf("%d$$$%s", regionID, node.Name),
				RegionCode: node.Name,
				Nodes:      []*tailcfg.DERPNode{node},
			}
			idx++
		}
	}
	return res
}

func (t *tester) Test(derpMap *tailcfg.DERPMap) *tailcfg.DERPMap {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	newDerpMap := derpMap.Clone()
	flattenedMap := flattenDEPRMap(derpMap)

	zap.L().Info("Start Test", zap.Int("regionCount", CountDERPMap(derpMap)))

	removeRegion := func(regionName string) {
		mu.Lock()
		defer mu.Unlock()
		res := strings.Split(regionName, "$$$")
		if len(res) != 2 {
			return
		}
		regionID, err := strconv.Atoi(res[0])
		if err != nil {
			return
		}
		nodeName := res[1]
		region := newDerpMap.Regions[regionID]
		if region == nil {
			return
		}
		for i, node := range region.Nodes {
			if node.Name == nodeName {
				region.Nodes = append(region.Nodes[:i], region.Nodes[i+1:]...)
				break
			}
		}
		if len(region.Nodes) == 0 {
			delete(newDerpMap.Regions, regionID)
		}
	}

	wg.Add(CountDERPMap(flattenedMap) + len(flattenedMap.Regions))
	ctx, cancel := context.WithCancel(t.ctx)

	for _, region := range flattenedMap.Regions {
		go func(region *tailcfg.DERPRegion) {
			defer wg.Done()
			latency, _, err := t.measureHTTPSLatency(ctx, region)
			if err != nil {
				zap.L().Debug("HTTPS Error", zap.Any("dest", region.RegionCode), zap.Error(err))
				removeRegion(region.RegionName)
			} else if latency > t.latencyLimit {
				zap.L().Debug("HTTPS Latency", zap.Any("dest", region.RegionCode), zap.Duration("latency", latency))
				removeRegion(region.RegionName)
			}
		}(region)
		for _, node := range region.Nodes {
			go func(node *tailcfg.DERPNode) {
				defer wg.Done()
				err := t.testDerpNode(ctx, node, false)
				if err != nil {
					zap.L().Debug("Node Error", zap.Any("dest", node.Name), zap.Error(err))
					removeRegion(node.Name)
				}
			}(node)
		}
	}

	wg.Wait()
	cancel()

	zap.L().Info("End Test", zap.Int("regionCount", CountDERPMap(newDerpMap)))

	return newDerpMap
}

func (t *tester) measureHTTPSLatency(ctx context.Context, reg *tailcfg.DERPRegion) (time.Duration, netip.Addr, error) {
	var result httpstat.Result
	ctx, cancel := context.WithTimeout(httpstat.WithHTTPStat(ctx, &result), t.probeTimeout)
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

func (t *tester) testDerpNode(ctx context.Context, derpNode *tailcfg.DERPNode, ipv6 bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.probeTimeout)
	defer cancel()

	var (
		dialer net.Dialer
	)

	checkSTUN4 := func(derpNode *tailcfg.DERPNode) error {
		u4, err := nettype.MakePacketListenerWithNetIP(netns.Listener(t.Logf, t.NetMon)).ListenPacket(ctx, "udp4", ":0")
		if err != nil {
			return fmt.Errorf("error creating IPv4 STUN listener: %v", err)
		}
		defer u4.Close()

		var addr netip.Addr
		if derpNode.IPv4 != "" {
			addr, err = netip.ParseAddr(derpNode.IPv4)
			if err != nil {
				// Error printed elsewhere
				return fmt.Errorf("error parsing node %q IPv4 address: %v", derpNode.HostName, err)
			}
		} else {
			addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", derpNode.HostName)
			if err != nil {
				return fmt.Errorf("error resolving node %q IPv4 addresses: %v", derpNode.HostName, err)
			}
			addr = addrs[0]
		}

		addrPort := netip.AddrPortFrom(addr, uint16(firstNonzero(derpNode.STUNPort, 3478)))

		txID := stun.NewTxID()
		req := stun.Request(txID)

		done := make(chan struct{})
		defer close(done)

		go func() {
			select {
			case <-ctx.Done():
			case <-done:
			}
			u4.Close()
		}()

		gotResponse := make(chan netip.AddrPort, 1)
		go func() {
			defer u4.Close()

			var buf [64 << 10]byte
			for {
				n, addr, err := u4.ReadFromUDPAddrPort(buf[:])
				if err != nil {
					return
				}
				pkt := buf[:n]
				if !stun.Is(pkt) {
					continue
				}
				ap := netaddr.Unmap(addr)
				if !ap.IsValid() {
					continue
				}
				tx, addrPort, err := stun.ParseResponse(pkt)
				if err != nil {
					continue
				}
				if tx == txID {
					gotResponse <- addrPort
					return
				}
			}
		}()

		_, err = u4.WriteToUDPAddrPort(req, addrPort)
		if err != nil {
			return fmt.Errorf("error sending IPv4 STUN packet to %v (%q): %v", addrPort, derpNode.HostName, err)
		}

		select {
		case <-gotResponse:
			return nil
		case <-ctx.Done():
			return fmt.Errorf("node %q did not return a IPv4 STUN response", derpNode.HostName)
		}
	}

	port := firstNonzero(derpNode.DERPPort, 443)

	var (
		v4Error error
		v6Error error
	)

	// Check IPv4 first
	addr := net.JoinHostPort(firstNonzero(derpNode.IPv4, derpNode.HostName), strconv.Itoa(port))
	conn, err := dialer.DialContext(ctx, "tcp4", addr)
	if err != nil {
		v4Error = fmt.Errorf("error connecting to node %q @ %q over IPv4: %w", derpNode.HostName, addr, err)
	} else {
		defer conn.Close()

		// Upgrade to TLS and verify that works properly.
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: firstNonzero(derpNode.CertName, derpNode.HostName),
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			v4Error = fmt.Errorf("error upgrading connection to node %q @ %q to TLS over IPv4: %w", derpNode.HostName, addr, err)
		}
	}

	if ipv6 {
		// Check IPv6
		addr = net.JoinHostPort(firstNonzero(derpNode.IPv6, derpNode.HostName), strconv.Itoa(port))
		conn, err = dialer.DialContext(ctx, "tcp6", addr)
		if err != nil {
			v6Error = fmt.Errorf("error connecting to node %q @ %q over IPv6: %w", derpNode.HostName, addr, err)
		} else {
			defer conn.Close()

			// Upgrade to TLS and verify that works properly.
			tlsConn := tls.Client(conn, &tls.Config{
				ServerName: firstNonzero(derpNode.CertName, derpNode.HostName),
				// TODO(andrew-d): we should print more
				// detailed failure information on if/why TLS
				// verification fails
			})
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				v6Error = fmt.Errorf("error upgrading connection to node %q @ %q to TLS over IPv6: %w", derpNode.HostName, addr, err)
			}
		}
	}

	if v4Error != nil && v6Error != nil {
		return fmt.Errorf("v4 Error: %v, v6 Error: %v", v4Error, v6Error)
	}

	return checkSTUN4(derpNode)
}

func firstNonzero[T comparable](items ...T) T {
	var zero T
	for _, item := range items {
		if item != zero {
			return item
		}
	}
	return zero
}
