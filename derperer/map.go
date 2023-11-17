package derperer

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"git.yoshino-s.xyz/yoshino-s/derperer/speedtest"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type DERPMapPolicy struct {
	// RecheckInterval is the interval to recheck a abandoned node.
	RecheckInterval time.Duration

	CheckDuration time.Duration

	TestConcurrency int

	BaselineBandwidth float64
}

type Map struct {
	*DERPMap

	policy       *DERPMapPolicy
	nextRegionID atomic.Int32
	logger       *zap.Logger

	testPool *pool.Pool
}

func NewMap(policy *DERPMapPolicy) *Map {
	return &Map{
		DERPMap:      NewDERPMap(),
		policy:       policy,
		nextRegionID: atomic.Int32{},
		logger:       zap.L(),
		testPool:     pool.New().WithMaxGoroutines(policy.TestConcurrency),
	}
}

func (d *Map) FilterDERPMap(status ...DERPRegionStatus) *DERPMap {
	newMap := &DERPMap{
		Regions:            map[int]*DERPRegion{},
		HomeParams:         d.HomeParams,
		OmitDefaultRegions: d.OmitDefaultRegions,
	}
	for _, region := range d.Regions {
		for _, s := range status {
			if region.Status == s {
				newMap.Regions[region.RegionID] = region
			}
		}
	}
	return newMap
}

func (d *Map) findByHostnameAndPort(hostname string, port ...int) *DERPRegion {
	for _, r := range d.Regions {
		for _, n := range r.Nodes {
			if n.HostName == hostname {
				if len(port) != 0 {
					p := n.DERPPort
					if p == 0 {
						p = 443
					}
					if p == port[0] {
						return r
					}
				} else {
					return r
				}
			}
		}
	}
	return nil
}

func (d *Map) testRegion(region *DERPRegion) {
	d.testPool.Go(func() {
		res, err := speedtest.CheckDerp(region.Convert(), d.policy.CheckDuration)
		if err != nil {
			d.logger.Error("failed to check derp", zap.Int("region_id", region.RegionID), zap.String("error", err.Error()))
			region.Error = err.Error()
			region.Status = DERPRegionStatusError
			return
		}
		region.Latency = res.Latency.String()
		region.Bandwidth = res.Bps.String()
		d.DERPMap.HomeParams.RegionScore[region.RegionID] = ((d.policy.BaselineBandwidth * 1024 * 1024) / res.Bps.Value())
		region.Status = DERPRegionStatusAlive
		d.logger.Debug("checked derp", zap.Int("region_id", region.RegionID), zap.String("bandwidth", res.Bps.String()), zap.String("latency", res.Latency.String()))
	})
}

func (d *Map) Recheck() {
	ticker := time.NewTicker(d.policy.RecheckInterval)
	for {
		select {
		case <-ticker.C:
			for _, region := range d.Regions {
				d.testPool.Go(func() {
					d.testRegion(region)
				})
			}
		}
	}
}

func (d *Map) buildNode(result fofa.FofaResult) (*DERPNode, error) {
	url, err := url.Parse(result.Host)
	if err != nil {
		return nil, err
	}

	host := url.Hostname()
	ip := result.IP
	port, err := strconv.Atoi(result.Port)
	if err != nil {
		return nil, err
	}

	node := &DERPNode{
		HostName: host,
		DERPPort: port,
	}

	if net.ParseIP(host) == nil {
		// resolve domain with both ipv4 and ipv6
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if ip.To4() != nil {
				node.IPv4 = ip.String()
			} else {
				node.IPv6 = ip.String()
			}
		}
	} else {
		node.InsecureForTests = true
		if net.ParseIP(ip).To4() != nil {
			node.IPv4 = ip
		} else {
			node.IPv6 = ip
		}
	}

	return node, nil
}

func (d *Map) AddFofaResult(result fofa.FofaResult) error {
	if result.Protocol != "https" {
		return nil
	}

	node, err := d.buildNode(result)
	if err != nil {
		return err
	}

	region := d.findByHostnameAndPort(node.HostName, node.DERPPort)
	if region != nil {
		return nil
	}

	regionID := d.nextRegionID.Load()
	d.nextRegionID.Add(1)

	code := result.Country
	if result.Region != "" {
		code += fmt.Sprintf("-%s", result.Region)
	}
	if result.City != "" {
		code += fmt.Sprintf("-%s", result.City)
	}
	if result.ASOrganization != "" {
		code += fmt.Sprintf("-%s", result.ASOrganization)
	}
	code += fmt.Sprintf("-%s", result.IP)

	node.RegionID = int(regionID)

	region = &DERPRegion{
		RegionName: code,
		RegionCode: code,
		RegionID:   int(regionID),
		Nodes:      []*DERPNode{node},
		Status:     DERPRegionStatusUnknown,
	}
	d.Regions[int(regionID)] = region

	d.testRegion(region)

	return nil
}
