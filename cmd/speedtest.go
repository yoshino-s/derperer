package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/yoshino-s/derperer/pkg/speedtest"
	"github.com/yoshino-s/go-framework/application"
	"github.com/yoshino-s/go-framework/configuration"
	"github.com/yoshino-s/go-framework/utils"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

var (
	speedtestApp = newSpeedTestCmdApp()
	speedtestCmd = &cobra.Command{
		Use: "speedtest",
		Run: func(cmd *cobra.Command, args []string) {
			app.Append(speedtest.New())
			app.Append(speedtestApp)

			app.Go(context.Background())
		},
	}
)

func init() {
	rootCmd.AddCommand(speedtestCmd)
	speedtestApp.Configuration().Register(speedtestCmd.Flags())
}

type speedTestCmdApp struct {
	*application.EmptyApplication
	config speedTestCmdConfig

	SpeedtestService *speedtest.SpeedTestService `inject:""`
}

type speedTestCmdConfig struct {
	DerpMapUrl   string        `mapstructure:"derp_map_url"`
	DerpRegionId int           `mapstructure:"derp_region_id"`
	Duration     time.Duration `mapstructure:"duration"`
}

func (s *speedTestCmdConfig) Read() {
	utils.MustDecodeFromMapstructure(viper.AllSettings(), s)
}

func (s *speedTestCmdConfig) Register(set *pflag.FlagSet) {
	set.String("derp_map_url", "https://controlplane.tailscale.com/derpmap/default", "derp map url")
	set.Int("derp_region_id", 0, "derp region id")
	set.Duration("duration", time.Second*30, "duration")
	utils.MustNoError(viper.BindPFlags(set))
	configuration.Register(s)
}

func newSpeedTestCmdApp() *speedTestCmdApp {
	return &speedTestCmdApp{
		EmptyApplication: application.NewEmptyApplication("speedTestCmdApp"),
	}
}

func (s *speedTestCmdApp) Configuration() configuration.Configuration {
	return &s.config
}

func (s *speedTestCmdApp) Run(ctx context.Context) {
	logger := s.Logger

	req, err := http.NewRequestWithContext(ctx, "GET", s.config.DerpMapUrl, nil)
	if err != nil {
		panic(errors.Errorf("create derp map request: %w", err))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(errors.Errorf("fetch derp map failed: %w", err))
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		panic(errors.Errorf("fetch derp map failed: %w", err))
	}
	if res.StatusCode != 200 {
		panic(errors.Errorf("fetch derp map: %v: %s", res.Status, b))
	}
	var dmap tailcfg.DERPMap
	if err = json.Unmarshal(b, &dmap); err != nil {
		panic(errors.Errorf("fetch DERP map: %w", err))
	}
	region := dmap.Regions[s.config.DerpRegionId]
	if region == nil {
		panic(errors.Errorf("derp region %d not found, vailable regions: %v", s.config.DerpRegionId, dmap.RegionIDs()))
	}
	logger.Info("derp region", zap.Any("region", region))

	if res, err := s.SpeedtestService.CheckDerp(region, s.config.Duration); err != nil {
		cobra.CheckErr(errors.Errorf("check derp: %w", err))
	} else {
		logger.Sugar().Infof("bandwidth: %s, totalBytes: %s, latency: %s", res.Bps.String(), res.TotalBytesSent.String(), res.Latency.String())
	}
}
