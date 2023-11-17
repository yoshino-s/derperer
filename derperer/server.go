package derperer

import (
	"context"
	"strings"
	"time"

	_ "git.yoshino-s.xyz/yoshino-s/derperer/docs"
	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/swagger"
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

	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Get("/", derperer.index)

	app.Get("/derp.json", derperer.getDerp)

	if config.AdminToken != "" {
		adminApi := app.Group("/admin", basicauth.New(basicauth.Config{
			Users: map[string]string{
				"admin": config.AdminToken,
			},
		}))

		adminApi.Get("/", derperer.adminIndex)

		adminApi.Get("/monitor", monitor.New())
		adminApi.Use(pprof.New(pprof.Config{
			Prefix: "/admin",
		}))
		adminApi.Get("/config", derperer.getConfig)
		adminApi.Post("/config", derperer.setConfig)
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

// @Summary Index
// @Produce html
// @Router / [get]
func (d *Derperer) index(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	return c.SendString(strings.TrimSpace(`
<a href="/derp.json">derp.json</a><br/>
<a href="/swagger/index.html">swagger</a><br/>
<a href="/admin">admin</a><br/>
		`))
}

// @Summary Get DERP Map
// @Param status query string false "alive|error|all" Enums(alive, error, all)
// @Param latency-limit query string false "latency limit, e.g. 500ms"
// @Param bandwidth-limit query string string "bandwidth limit, e.g. 2Mbps"
// @Produce json
// @Router /derp.json [get]
func (d *Derperer) getDerp(c *fiber.Ctx) error {
	var filter DERPMapFilter
	if err := c.QueryParser(&filter); err != nil {
		return err
	}
	m, err := d.derpMap.FilterDERPMap(filter)
	if err != nil {
		return err
	}
	return c.JSON(m)
}

// @securityDefinitions.basic BasicAuth

// @Summary Admin Index
// @Produce html
// @Security BasicAuth
// @Router /admin [get]
func (d *Derperer) adminIndex(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	return c.SendString(`
	<a href="/admin/monitor">monitor</a><br/>
	<a href="/admin/debug/pprof">pprof</a><br/>
	<a href="/admin/config">config</a> or <code>POST</code> to change config <br/>
	`)
}

// @Summary Get Server Config
// @Produce json
// @Security BasicAuth
// @Router /admin/config [get]
func (d *Derperer) getConfig(c *fiber.Ctx) error {
	return c.JSON(d.config)
}

// @Summary Change Server Config
// @Accept json
// @Param config body derperer.DerpererConfig true "config"
// @Produce json
// @Security BasicAuth
// @Router /admin/config [post]
func (d *Derperer) setConfig(c *fiber.Ctx) error {
	if err := c.BodyParser(&d.config); err != nil {
		return err
	}
	return c.JSON(d.config)
}
