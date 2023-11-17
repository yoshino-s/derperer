package derperer

import (
	"context"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/sourcegraph/conc"
	"go.uber.org/zap"
)

const FINGERPRINT = `"<h1>DERP</h1>"`

type Derperer struct {
	config  DerpererConfig
	app     *fiber.App
	ctx     context.Context
	derpMap *Map
}

type DerpererConfig struct {
	Address       string
	AdminToken    string
	FetchInterval time.Duration
	FetchBatch    int
	FofaClient    fofa.Fofa
	DERPMapPolicy DERPMapPolicy
}

func NewDerperer(config DerpererConfig) (*Derperer, error) {
	app := fiber.New()
	ctx := context.Background()
	derperer := &Derperer{
		config:  config,
		app:     app,
		ctx:     ctx,
		derpMap: NewMap(&config.DERPMapPolicy),
	}

	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString(`
			<a href="/derp.json">derp.json</a>?status=[<b>alive</b>|abandoned|error|all],...<br/>
			<a href="/admin">admin</a><br/>
			`)
	})

	app.Get("/derp.json", func(c *fiber.Ctx) error {
		status := c.Query("status")
		if status == "all" {
			return c.JSON(derperer.derpMap.DERPMap)
		}
		if status == "" {
			status = "alive"
		}
		s, err := ParseDERPRegionStatus(status)
		if err != nil {
			return err
		}
		return c.JSON(derperer.derpMap.FilterDERPMap(s))
	})
	if config.AdminToken != "" {
		adminApi := app.Group("/admin", basicauth.New(basicauth.Config{
			Users: map[string]string{
				"admin": config.AdminToken,
			},
		}))

		adminApi.Get("/", func(c *fiber.Ctx) error {
			c.Set("Content-Type", "text/html")
			return c.SendString(`
			<a href="/admin/monitor">monitor</a><br/>
			<a href="/admin/debug/pprof">pprof</a><br/>
			<a href="/admin/config">config</a> or <code>POST</code> to change config <br/>
			`)
		})

		adminApi.Get("/monitor", monitor.New())
		adminApi.Use(pprof.New(pprof.Config{
			Prefix: "/admin",
		}))
		adminApi.Get("/config", func(c *fiber.Ctx) error {
			return c.JSON(config)
		})
		adminApi.Post("/config", func(c *fiber.Ctx) error {
			if err := c.BodyParser(&config); err != nil {
				return err
			}
			return c.JSON(config)
		})
	}

	return derperer, nil
}

func (d *Derperer) FetchFofaData() {
	logger := zap.L()
	logger.Info("fetching fofa")
	res, finish, err := d.config.FofaClient.Query(FINGERPRINT, d.config.FetchBatch, -1)
	if err != nil {
		logger.Error("failed to query fofa", zap.Error(err))
	}
	for {
		select {
		case r := <-res:
			d.derpMap.AddFofaResult(r)
		case <-finish:
			logger.Info("fofa query finished")
			return
		}
	}
}

func (d *Derperer) Start() {
	wg := conc.WaitGroup{}

	wg.Go(d.derpMap.Recheck)

	wg.Go(func() {
		for {
			d.FetchFofaData()
			time.Sleep(d.config.FetchInterval)
		}
	})

	wg.Go(func() {
		d.app.Listen(d.config.Address)
	})

	wg.Wait()
}
