package derperer

import (
	"time"

	"github.com/yoshino-s/derperer/pkg/speedtest"
	"tailscale.com/tailcfg"
)

type DerpStatus string

const (
	DerpStatusUnknown   DerpStatus = "unknown"
	DerpStatusAvailable DerpStatus = "available"
	DerpStatusError     DerpStatus = "error"
)

type DerpEndpoint struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`

	Region   string `json:"region"`
	Host     string `json:"host"`
	IPv4     string `json:"ipv4,omitempty"`
	IPv6     string `json:"ipv6,omitempty"`
	Port     int    `json:"port,omitempty"`
	Insecure bool   `json:"insecure_for_tests,omitempty"`

	Status    DerpStatus     `json:"status"`
	Latency   time.Duration  `json:"latency,omitempty"`
	Bandwidth speedtest.Unit `json:"bandwidth,omitempty"`
	Error     string         `json:"error,omitempty"`
}

func (d *DerpEndpoint) Convert() *DERPRegion {
	return &DERPRegion{
		DERPRegion: tailcfg.DERPRegion{
			RegionID:   d.ID,
			RegionCode: d.Name,
			RegionName: d.Name,
		},
		Nodes: []*DERPNode{
			{
				DERPNode: tailcfg.DERPNode{
					Name:             d.Name,
					RegionID:         d.ID,
					HostName:         d.Host,
					IPv4:             d.IPv4,
					IPv6:             d.IPv6,
					DERPPort:         d.Port,
					InsecureForTests: d.Insecure,
				},
				Latency:   d.Latency,
				Bandwidth: d.Bandwidth,
				Status:    d.Status,
			},
		},
	}
}

type DerpEndpoints []*DerpEndpoint

func (d DerpEndpoints) Len() int { return len(d) }

func (d DerpEndpoints) Convert() *DERPMap {
	if d == nil {
		return &DERPMap{
			DERPMap: tailcfg.DERPMap{
				Regions: map[int]*tailcfg.DERPRegion{},
			},
		}
	}
	m := &DERPMap{
		DERPMap: tailcfg.DERPMap{
			HomeParams: &tailcfg.DERPHomeParams{
				RegionScore: map[int]float64{},
			},
		},
		Regions: make(map[int]*DERPRegion),
	}
	for _, endpoint := range d {
		m.Regions[endpoint.ID] = endpoint.Convert()
		m.HomeParams.RegionScore[endpoint.ID] = (endpoint.Bandwidth.Value / (1000 * 1024 * 1024))
	}
	return m
}

type DerpQueryParams struct {
	Status         DerpStatus    `query:"status" json:"status" enums:"alive,error,all"`
	LatencyLimit   time.Duration `query:"latency-limit" json:"latency_limit"`
	BandwidthLimit float64       `query:"bandwidth-limit" json:"bandwidth_limit"`
}

func (d DerpEndpoints) Query(params *DerpQueryParams) DerpEndpoints {
	var res DerpEndpoints
	for _, endpoint := range d {
		if params.Status != "" && endpoint.Status != params.Status {
			continue
		}
		if params.LatencyLimit != 0 && endpoint.Latency > params.LatencyLimit {
			continue
		}
		if params.BandwidthLimit != 0 && endpoint.Bandwidth.Value < params.BandwidthLimit {
			continue
		}
		res = append(res, endpoint)
	}
	return res
}

func (d DerpEndpoints) Exist(host string, port int) (*DerpEndpoint, bool) {
	for _, endpoint := range d {
		if endpoint.Host == host && endpoint.Port == port {
			return endpoint, true
		}
	}
	return nil, false
}
