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
					Name:     "199.38.181.104",
					RegionID: 1,
					HostName: "derp1f.tailscale.com",
					IPv4:     "199.38.181.104",
				},
			},
		},
	},
}

func TestDebugDERPNode(t *testing.T) {
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(zap.AddCaller()))
	zap.ReplaceGlobals(logger)
	server, err := derperer.NewDerperer(derperer.DerpererConfig{
		LatencyLimit: 100 * time.Second,
		ProbeTimeout: 10 * time.Second,
		DatabaseUri:  "mongodb://derperer:derperer@mongodb.storage",
	})
	assert.NoError(t, err)

	result, _ := server.Test(derpMap)

	assert.NoError(t, err)
	assert.Equal(t, 1, derperer.CountDERPMap(result))
}
