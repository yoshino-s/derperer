package derperer

import (
	"encoding/binary"
	"net"
	"net/url"
	"strconv"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"tailscale.com/tailcfg"
)

func Convert(result []fofa.FofaResult) (tailcfg.DERPMap, error) {
	derpMap := tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}

	for _, r := range result {
		ip := net.ParseIP(r.IP).To4()
		if ip == nil {
			continue
		}

		host, err := url.Parse(r.Host)
		if err != nil {
			continue
		}

		port, err := strconv.Atoi(r.Port)
		if err != nil {
			continue
		}

		regionID := int(binary.BigEndian.Uint32(ip))
		derpMap.Regions[regionID] = &tailcfg.DERPRegion{
			RegionID:   regionID,
			RegionName: r.IP,
			Nodes: []*tailcfg.DERPNode{
				{
					Name:             r.IP,
					RegionID:         regionID,
					HostName:         host.Hostname(),
					IPv4:             r.IP,
					InsecureForTests: true,
					DERPPort:         port,
				},
			},
		}
	}
	return derpMap, nil
}
