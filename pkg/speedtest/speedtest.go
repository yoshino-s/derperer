package speedtest

import (
	"fmt"
	"time"

	"github.com/go-errors/errors"
	"github.com/yoshino-s/go-framework/application"
	"go.uber.org/zap"
	"tailscale.com/derp"
	"tailscale.com/derp/derphttp"
	"tailscale.com/net/netmon"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

type SpeedTestService struct {
	*application.EmptyApplication
}

func New() *SpeedTestService {
	return &SpeedTestService{
		EmptyApplication: application.NewEmptyApplication("SpeedTestService"),
	}
}

func (s *SpeedTestService) CheckDerp(region *tailcfg.DERPRegion, duration time.Duration) (*SpeedTestResult, error) {
	var err error
	var m derp.ReceivedMessage
	logger := s.Logger

	getRegion := func() *tailcfg.DERPRegion {
		return region
	}

	priv1 := key.NewNode()
	priv2 := key.NewNode()

	c1 := derphttp.NewRegionClient(priv1, logger.Sugar().Debugf, netmon.NewStatic(), getRegion)
	c2 := derphttp.NewRegionClient(priv2, logger.Sugar().Debugf, netmon.NewStatic(), getRegion)

	defer c1.Close()
	defer c2.Close()

	c2.NotePreferred(true) // just to open it

	m, err = c2.Recv()
	if err != nil {
		return nil, err
	}
	info, ok := m.(derp.ServerInfoMessage)
	if !ok {
		return nil, errors.Errorf("got %T, want derp.ServerInfoMessage", m)
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

	return s.measure(c1, c2, priv2.Public(), duration)
}
