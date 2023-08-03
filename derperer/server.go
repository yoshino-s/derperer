package derperer

import (
	"context"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"
	"tailscale.com/net/netmon"
	"tailscale.com/tailcfg"
)

type Derperer struct {
	DerpererConfig
	*tester
	app        *iris.Application
	rawResults [][]fofa.FofaResult
	derpMap    tailcfg.DERPMap
	netMon     *netmon.Monitor
	ctx        context.Context
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
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	t, err := newTester(
		ctx,
		zap.L().Sugar().Infof,
		config.LatencyLimit,
	)
	if err != nil {
		return nil, err
	}
	derperer := &Derperer{
		DerpererConfig: config,
		tester:         t,
		app:            app,
		rawResults:     [][]fofa.FofaResult{},
		derpMap:        tailcfg.DERPMap{},
		ctx:            ctx,
	}

	app.Get("/derp.json", derperer.getDerpMap)

	return derperer, nil
}

func (d *Derperer) FetchFofaData() {
	zap.L().Info("fetching fofa")
	rawResults := [][]fofa.FofaResult{}
	res, finish, err := d.FofaClient.Query("fid=\"QSk7WHdA/IWH9oZf9xszuw==\"", d.FetchBatch, -1)
	if err != nil {
		zap.L().Error("failed to query fofa", zap.Error(err))
	}
	total := 0
	func() {
		for {
			select {
			case r := <-res:
				rawResults = append(rawResults, r)
				total += len(r)
			case <-finish:
				return
			}
		}
	}()
	zap.L().Info("fetched fofa", zap.Int("result_count", total))
	d.rawResults = rawResults
}

func (d *Derperer) UpdateDERPMap() {
	fullDERPMap := tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}
	for _, rawResult := range d.rawResults {
		derpMap, err := Convert(rawResult)
		if err != nil {
			zap.L().Error("failed to convert", zap.Error(err))
			continue
		}
		newDerpMap := d.Test(&derpMap)
		for regionID, region := range newDerpMap.Regions {
			if _, ok := fullDERPMap.Regions[regionID]; !ok {
				fullDERPMap.Regions[regionID] = region
			} else {
				fullDERPMap.Regions[regionID].Nodes = append(fullDERPMap.Regions[regionID].Nodes, region.Nodes...)
			}
		}

	}
	d.derpMap = fullDERPMap
	zap.L().Info("updated derp map", zap.Int("region_count", len(fullDERPMap.Regions)))
}

func (d *Derperer) Start() error {

	go func() {
		for {
			d.FetchFofaData()
			d.UpdateDERPMap()
			time.Sleep(d.FetchInterval)
		}
	}()

	go func() {
		for {
			d.UpdateDERPMap()
			time.Sleep(d.UpdateInterval)
		}
	}()

	return d.app.Listen(d.Address)
}

func (d *Derperer) getDerpMap(ctx iris.Context) {
	ctx.JSON(d.derpMap)
}
