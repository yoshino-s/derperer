package derperer

import (
	"context"
	"sync"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

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
	FetchBatch     int
}

func NewDerperer(config DerpererConfig) (*Derperer, error) {
	app := iris.New()
	ctx := context.Background()
	t, err := newTester(
		ctx,
		zap.L().Sugar().Infof,
		config.LatencyLimit,
		10*time.Second,
		2*time.Second,
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
	res, finish, err := d.FofaClient.Query(`body="<a href=\"https://pkg.go.dev/tailscale.com/derp\">DERP</a>"`, d.FetchBatch, -1)
	if err != nil {
		zap.L().Error("failed to query fofa", zap.Error(err))
	}
	func() {
		for {
			select {
			case r := <-res:
				zap.L().Info("fetched fofa", zap.Int("count", len(r)))
				d.UpdateDERPMap(r)
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

func (d *Derperer) getDerpMap(ctx iris.Context) {
	ctx.JSON(d.derpMap)
}
