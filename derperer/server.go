package derperer

import (
	"context"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

type Derperer struct {
	DerpererConfig
	app        *iris.Application
	rawResults [][]fofa.FofaResult
	derpMap    tailcfg.DERPMap
}

type DerpererConfig struct {
	Address        string
	UpdateInterval time.Duration
	FetchInterval  time.Duration
	FofaClient     fofa.Fofa
	LatencyLimit   time.Duration
	FetchBatch     int
}

func NewDerperer(config DerpererConfig) *Derperer {
	app := iris.New()
	derperer := &Derperer{
		DerpererConfig: config,
		app:            app,
		rawResults:     [][]fofa.FofaResult{},
		derpMap:        tailcfg.DERPMap{},
	}

	app.Get("/derp.json", derperer.getDerpMap)

	return derperer
}

func (d *Derperer) Start() error {
	client := &Client{
		Logf:  zap.L().Sugar().Infof,
		VLogf: zap.L().Sugar().Debugf,
	}

	fetchFofaData := func() {
		zap.L().Info("fetching fofa")
		rawResults := [][]fofa.FofaResult{}
		res, finish, err := d.FofaClient.Query("fid=\"QSk7WHdA/IWH9oZf9xszuw==\"", d.FetchBatch, -1)
		if err != nil {
			zap.L().Error("failed to query fofa", zap.Error(err))
		}
		func() {
			for {
				select {
				case r := <-res:
					rawResults = append(rawResults, r)
				case <-finish:
					return
				}
			}
		}()
		zap.L().Info("fetched fofa", zap.Int("result_count", len(rawResults)))
		d.rawResults = rawResults
	}

	updateDERPMap := func() {
		zap.L().Info("updating derp map")
		fullDERPMap := tailcfg.DERPMap{
			Regions: map[int]*tailcfg.DERPRegion{},
		}
		for _, rawResult := range d.rawResults {
			derpMap, err := Convert(rawResult)
			if err != nil {
				zap.L().Error("failed to convert", zap.Error(err))
				continue
			}
			report, err := client.GetReport(context.Background(), &derpMap)
			if err != nil {
				zap.L().Error("failed to get report", zap.Error(err))
				continue
			}
			for regionID, region := range derpMap.Regions {
				if report.RegionLatency[regionID] > d.LatencyLimit {
					region.Avoid = true
				}
				fullDERPMap.Regions[regionID] = region
			}
		}
		d.derpMap = fullDERPMap
		zap.L().Info("updated derp map", zap.Int("region_count", len(fullDERPMap.Regions)))
	}

	go func() {
		for {
			fetchFofaData()
			updateDERPMap()
			time.Sleep(d.FetchInterval)
		}
	}()

	go func() {
		for {
			updateDERPMap()
			time.Sleep(d.UpdateInterval)
		}
	}()

	return d.app.Listen(d.Address)
}

func (d *Derperer) getDerpMap(ctx iris.Context) {
	ctx.JSON(d.derpMap)
}
