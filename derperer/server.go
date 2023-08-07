package derperer

import (
	"context"
	"strings"
	"sync"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

const FINGERPRINT = `"<h1>DERP</h1>" && cert.is_valid=true && cert.is_match=true && is_domain=true`

type Derperer struct {
	DerpererConfig
	*tester
	app     *iris.Application
	derpMap *tailcfg.DERPMap
	ctx     context.Context
	mu      *sync.Mutex
}

type DerpererConfig struct {
	Address        string
	UpdateInterval time.Duration
	FetchInterval  time.Duration
	FofaClient     fofa.Fofa
	LatencyLimit   time.Duration
	ProbeTimeout   time.Duration
	FetchBatch     int
	TestBatch      int
}

func NewDerperer(config DerpererConfig) (*Derperer, error) {
	app := iris.New()
	ctx := context.Background()
	t, err := newTester(
		ctx,
		zap.L().Sugar().Infof,
		config.LatencyLimit,
		config.ProbeTimeout,
	)
	if err != nil {
		return nil, err
	}
	derperer := &Derperer{
		DerpererConfig: config,
		tester:         t,
		app:            app,
		derpMap: &tailcfg.DERPMap{
			Regions: map[int]*tailcfg.DERPRegion{},
		},
		ctx: ctx,
		mu:  &sync.Mutex{},
	}

	app.Get("/derp.json", derperer.getDerpMap)

	return derperer, nil
}

func (d *Derperer) FetchFofaData() {
	zap.L().Info("fetching fofa")
	res, finish, err := d.FofaClient.Query(FINGERPRINT, d.FetchBatch, -1)
	if err != nil {
		zap.L().Error("failed to query fofa", zap.Error(err))
	}
	buf := make([]fofa.FofaResult, 0, d.TestBatch)
	func() {
		for {
			select {
			case r := <-res:
				buf = append(buf, r)
				if len(buf) == d.TestBatch {
					d.UpdateDERPMap(buf)
					buf = make([]fofa.FofaResult, 0, d.TestBatch)
				}
			case <-finish:
				return
			}
		}
	}()
}

func (d *Derperer) UpdateDERPMap(rawResult []fofa.FofaResult) {
	derpMap, err := Convert(rawResult)
	if err != nil {
		zap.L().Error("failed to convert", zap.Error(err))
		return
	}
	newDerpMap := d.Test(derpMap)

	d.mu.Lock()
	for regionID, region := range newDerpMap.Regions {
		if _, ok := d.derpMap.Regions[regionID]; !ok {
			d.derpMap.Regions[regionID] = region
		} else {
			d.derpMap.Regions[regionID].Nodes = append(d.derpMap.Regions[regionID].Nodes, region.Nodes...)
		}
	}
	d.mu.Unlock()
}

func (d *Derperer) Start() error {
	go func() {
		for {
			d.mu.Lock()
			derpMap := d.derpMap.Clone()
			d.derpMap = d.Test(derpMap)
			d.mu.Unlock()
			time.Sleep(d.UpdateInterval)
		}
	}()

	go func() {
		for {
			d.FetchFofaData()
			time.Sleep(d.FetchInterval)
		}
	}()

	return d.app.Listen(d.Address)
}

func contain(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func (d *Derperer) getDerpMap(ctx iris.Context) {
	nodeName := strings.Split(ctx.URLParam("node"), ";")
	if ctx.URLParam("node") != "" {
		derpMap := &tailcfg.DERPMap{
			Regions: map[int]*tailcfg.DERPRegion{},
		}
		for regionID, region := range d.derpMap.Regions {
			for _, node := range region.Nodes {
				if contain(nodeName, node.Name) {
					if _, ok := derpMap.Regions[regionID]; !ok {
						derpMap.Regions[regionID] = &tailcfg.DERPRegion{
							RegionID: regionID,
							Nodes:    []*tailcfg.DERPNode{},
						}
					}
					derpMap.Regions[regionID].Nodes = append(derpMap.Regions[regionID].Nodes, node)
				}
			}
		}
		ctx.JSON(derpMap)
	} else {
		ctx.JSON(d.derpMap)
	}
}
