package speedtest

import (
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"
	"tailscale.com/derp"
	"tailscale.com/derp/derphttp"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

func CheckDerp(region *tailcfg.DERPRegion, duration time.Duration) (*BandWidthResult, error) {
	var err error
	logger := zap.L()
	var m derp.ReceivedMessage

	getRegion := func() *tailcfg.DERPRegion {
		return region
	}

	priv1 := key.NewNode()
	priv2 := key.NewNode()

	c1 := derphttp.NewRegionClient(priv1, log.Printf, nil, getRegion)
	c2 := derphttp.NewRegionClient(priv2, log.Printf, nil, getRegion)

	defer c1.Close()
	defer c2.Close()

	c2.NotePreferred(true) // just to open it

	m, err = c2.Recv()
	if err != nil {
		return nil, err
	}
	info, ok := m.(derp.ServerInfoMessage)
	if !ok {
		return nil, fmt.Errorf("got %T, want derp.ServerInfoMessage", m)
	}
	logger.Debug("c1 got ServerInfoMessage", zap.Any("info", info))

	m, err = c1.Recv()
	if err != nil {
		return nil, err
	}
	info, ok = m.(derp.ServerInfoMessage)
	if !ok {
		return nil, fmt.Errorf("got %T, want derp.ServerInfoMessage", m)
	}
	logger.Debug("c2 got ServerInfoMessage", zap.Any("info", info))

	return measure(c1, c2, priv2.Public(), duration)
}
