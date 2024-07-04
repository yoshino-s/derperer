package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/yoshino-s/derperer/speedtest"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

var (
	derpMapUrl   string
	derpRegionId string
	duration     time.Duration
)

var speedtestCmd = &cobra.Command{
	Use: "speedtest",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()
		req, err := http.NewRequestWithContext(ctx, "GET", derpMapUrl, nil)
		if err != nil {
			cobra.CheckErr(fmt.Errorf("create derp map request: %w", err))
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			cobra.CheckErr(fmt.Errorf("fetch derp map failed: %w", err))
		}
		defer res.Body.Close()
		b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		if err != nil {
			cobra.CheckErr(fmt.Errorf("fetch derp map failed: %w", err))
		}
		if res.StatusCode != 200 {
			cobra.CheckErr(fmt.Errorf("fetch derp map: %v: %s", res.Status, b))
		}
		var dmap tailcfg.DERPMap
		if err = json.Unmarshal(b, &dmap); err != nil {
			cobra.CheckErr(fmt.Errorf("fetch DERP map: %w", err))
		}
		var region *tailcfg.DERPRegion
		for _, r := range dmap.Regions {
			if r.RegionCode == derpRegionId {
				region = r
			}
		}
		if region == nil {
			for _, r := range dmap.Regions {
				log.Printf("Known region: %q", r.RegionCode)
			}
			log.Fatalf("unknown region %q", derpRegionId)
			panic("unreachable")
		}
		if res, err := speedtest.CheckDerp(region, duration); err != nil {
			cobra.CheckErr(fmt.Errorf("check derp: %w", err))
		} else {
			zap.L().Sugar().Infof("bandwidth: %s, totalBytes: %s, latency: %s", res.Bps.String(), res.TotalBytesSent.String(), res.Latency.String())
		}
	},
}

func init() {
	rootCmd.AddCommand(speedtestCmd)

	speedtestCmd.Flags().StringVarP(&derpMapUrl, "derpMapUrl", "u", "https://controlplane.tailscale.com/derpmap/default", "derp map url")
	speedtestCmd.Flags().StringVarP(&derpRegionId, "derpRegionId", "r", "", "derp region id")
	speedtestCmd.Flags().DurationVarP(&duration, "duration", "d", time.Second*30, "duration")
}
