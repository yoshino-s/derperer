package http

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/yoshino-s/derperer/internal/derperer"
	"github.com/yoshino-s/go-framework/application"
	"github.com/yoshino-s/go-framework/handlers/http"

	_ "embed"

	_ "github.com/yoshino-s/derperer/internal/handler/http/docs"

	echoSwagger "github.com/swaggo/echo-swagger"
)

var _ application.Application = (*Handler)(nil)

type Handler struct {
	*http.Handler

	Derperer *derperer.DerpererService `inject:""`
}

func New() *Handler {
	return &Handler{
		Handler: http.New(),
	}
}

func (h *Handler) Setup(ctx context.Context) {
	h.Handler.Setup(ctx)
	h.GET("/", echo.HandlerFunc(h.index))
	h.GET("/derp.json", echo.HandlerFunc(h.getDerp))
	h.GET("/swagger/*", echoSwagger.WrapHandler)
}

func (h *Handler) Run(ctx context.Context) {
	h.Handler.Run(ctx)
}

//go:embed index.html
var indexHTMLContent string

// @Summary Index
// @Produce html
// @Router / [get]
func (h *Handler) index(c echo.Context) error {
	return c.HTML(200, indexHTMLContent)
}

// @Summary Get DERP Map
// @Param status query string false "alive|error|all" Enums(alive, error, all)
// @Param latency-limit query string false "latency limit, e.g. 500ms"
// @Param bandwidth-limit query string string "bandwidth limit, e.g. 2Mbps"
// @Produce json
// @Router /derp.json [get]
func (h *Handler) getDerp(c echo.Context) error {
	// var filter DERPMapFilter
	// if err := c.QueryParser(&filter); err != nil {
	// 	return err
	// }
	// m, err := d.derpMap.FilterDERPMap(filter)
	// if err != nil {
	// 	return err
	// }
	// return c.JSON(m)
	var query derperer.DerpQueryParams

	c.Bind(&query)

	m := h.Derperer.DerpEndpoints.Query(&query).Convert()

	return c.JSON(200, m)
}
