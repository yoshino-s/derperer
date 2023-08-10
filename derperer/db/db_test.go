package db_test

import (
	"context"
	"testing"

	"git.yoshino-s.xyz/yoshino-s/derperer/derperer/db"
	"github.com/stretchr/testify/assert"
	"tailscale.com/tailcfg"
)

var derpMap = &tailcfg.DERPMap{
	Regions: map[int]*tailcfg.DERPRegion{
		1: {
			RegionID:   1,
			RegionName: "test",
			Nodes: []*tailcfg.DERPNode{
				{
					Name:     "derp1.webrtc.win",
					RegionID: 1,
					HostName: "https://derp.anxincloud.cn",
					IPv4:     "123.249.97.214",
					DERPPort: 443,
				},
			},
		},
	},
}

func TestDB(t *testing.T) {
	db, err := db.New(context.TODO(), "mongodb://derperer:derperer@mongodb.storage")
	assert.NoError(t, err)

	err = db.Drop()
	assert.NoError(t, err)

	err = db.InsertDERPRegion(derpMap.Regions[1])
	assert.NoError(t, err)

	err = db.InsertDERPRegion(derpMap.Regions[1])
	assert.NoError(t, err)

	derpMap, err := db.GetDERPMap()
	assert.NoError(t, err)
	assert.Equal(t, len(derpMap.Regions), 1)
	assert.Equal(t, derpMap.Regions[114000].RegionName, "test")

	count, err := db.BanRegion(114000)
	assert.NoError(t, err)
	assert.Equal(t, count, 1)

	derpMap, err = db.GetDERPMap()
	assert.NoError(t, err)
	assert.Equal(t, len(derpMap.Regions), 0)
}
