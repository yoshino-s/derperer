package derperer_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"git.yoshino-s.xyz/yoshino-s/derperer/derperer"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"tailscale.com/tailcfg"
)

var derpMap = &tailcfg.DERPMap{
	Regions: map[int]*tailcfg.DERPRegion{
		1: {
			RegionID:   1,
			RegionName: "test",
			Nodes: []*tailcfg.DERPNode{
				{
					Name:     "tailscale.jvmkit.com",
					RegionID: 1,
					HostName: "tailscale.jvmkit.com",
					IPv4:     "78.46.187.22",
					DERPPort: 443,
				},
			},
		},
	},
}

func TestDebugDERPNode(t *testing.T) {
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(zap.AddCaller()))
	zap.ReplaceGlobals(logger)
	server, err := derperer.NewDerperer(derperer.DerpererConfig{
		LatencyLimit: time.Second,
	})
	assert.NoError(t, err)

	result := server.Test(derpMap)

	assert.NoError(t, err)
	zap.L().Info("Result", zap.Any("result", result))
}
