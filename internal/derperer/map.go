package derperer

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc"
	"github.com/sourcegraph/conc/pool"
	"github.com/yoshino-s/derperer/pkg/speedtest"
	"github.com/yoshino-s/go-app/fofa"
	"github.com/yoshino-s/go-framework/application"
	"github.com/yoshino-s/go-framework/configuration"
	"go.uber.org/zap"
)

type DerpererService struct {
	*application.EmptyApplication
	config config

	DerpEndpoints DerpEndpoints

	nextRegionID *atomic.Int32

	SpeedtestService *speedtest.SpeedTestService `inject:""`
	Fofa             *fofa.FofaApp               `inject:""`
}

func New() *DerpererService {
	nextRegionID := &atomic.Int32{}
	nextRegionID.Store(900)

	return &DerpererService{
		EmptyApplication: application.NewEmptyApplication("Derperer"),
		nextRegionID:     nextRegionID,
	}
}

func (d *DerpererService) Configuration() configuration.Configuration {
	return &d.config
}

func (d *DerpererService) testDerpEndpoint(endpoint *DerpEndpoint) {
	res, err := d.SpeedtestService.CheckDerp(endpoint.Convert(), d.config.CheckDuration)
	if err != nil {
		d.Logger.Error("failed to check derp", zap.Any("endpoint", endpoint), zap.Error(err))
		endpoint.Error = err.Error()
		endpoint.Status = DerpStatusError
	} else {
		endpoint.Latency = res.Latency
		endpoint.Bandwidth = res.Bps
		endpoint.Status = DerpStatusAvailable
		d.Logger.Debug("checked derp", zap.Any("endpoint", endpoint))
	}
}

func (d *DerpererService) Run(ctx context.Context) {
	wg := conc.NewWaitGroup()
	wg.Go(func() { d.recheck(ctx) })
	wg.Go(func() { d.refetch(ctx) })

	wg.Wait()
}

const FINGERPRINT = `body="<h1>DERP</h1>"`
const FINGERPRIINT_CN = `body="<h1>DERP</h1>" && country="CN"`

func (d *DerpererService) refetch(ctx context.Context) {
	t := time.After(0)
	for {
		select {
		case <-t:
			page := 1
			count := 0
			for {
				if count > d.config.FetchLimit {
					d.Logger.Debug("reach fetch limit")
					break
				}
				d.Logger.Debug("querying fofa", zap.Int("page", page))
				fingerprint := FINGERPRINT
				if d.config.CN {
					fingerprint = FINGERPRIINT_CN
				}
				res, err := d.Fofa.Query(fingerprint, page, 100, fofa.WithExtraFields(
					"country", "region", "city", "as_organization",
				))
				if err != nil {
					d.Logger.Error("failed to fetch derp endpoints from fofa", zap.Error(err))
					break
				}
				for _, asset := range res {
					d.addDerpEndpoint(asset)
					count++
				}
				page++
			}
			t = time.After(d.config.RefetchInterval)
		case <-ctx.Done():
			return
		}
	}
}

func (d *DerpererService) recheck(ctx context.Context) {
	t := time.After(0)
	for {
		select {
		case <-t:
			d.Logger.Debug("start recheck")
			pool := pool.New().WithMaxGoroutines(d.config.CheckConcurrency)
			for _, endpoint := range d.DerpEndpoints {
				pool.Go(func() {
					d.testDerpEndpoint(endpoint)
				})
			}
			pool.Wait()
			t = time.After(d.config.RecheckInterval)
		case <-ctx.Done():
			return
		}
	}
}

func (m *DerpererService) addDerpEndpoint(asset fofa.Asset) (*DerpEndpoint, error) {
	host := asset.URL.Hostname()
	port, err := strconv.Atoi(asset.URL.Port())
	if err != nil {
		return nil, err
	}

	if exist, ok := m.DerpEndpoints.Exist(host, port); ok {
		return exist, nil
	}

	node := &DerpEndpoint{
		Host: host,
		Port: port,
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
		node.Insecure = true
		if asset.IP.To4() != nil {
			node.IPv4 = asset.IP.String()
		} else {
			node.IPv6 = asset.IP.String()
		}
	}

	regionID := m.nextRegionID.Load()
	m.nextRegionID.Add(1)

	code := asset.Raw["country"]
	if asset.Raw["region"] != "" {
		code += fmt.Sprintf("-%s", asset.Raw["region"])
	}
	if asset.Raw["city"] != "" {
		code += fmt.Sprintf("-%s", asset.Raw["city"])
	}
	if asset.Raw["as_organization"] != "" {
		code += fmt.Sprintf("-%s", asset.Raw["as_organization"])
	}
	code += fmt.Sprintf("-%s", asset.Raw["ip"])
	node.Name = code

	node.ID = int(regionID)

	m.DerpEndpoints = append(m.DerpEndpoints, node)
	return node, nil
}
