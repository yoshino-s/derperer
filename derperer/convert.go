package derperer

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

func CountDERPMap(derpMap *tailcfg.DERPMap) int {
	count := 0
	for _, region := range derpMap.Regions {
		count += len(region.Nodes)
	}
	return count
}

func Convert(result []fofa.FofaResult) (*tailcfg.DERPMap, error) {
	derpMap := &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}

	var id = 0

	for _, r := range result {
		if r.Protocol != "https" {
			continue
		}

		if net.ParseIP(r.IP).To4() == nil {
			continue
		}

		regionID := id
		id++

		node := &tailcfg.DERPNode{
			// Name:     nodeName,
			RegionID: regionID,
			// HostName: host,
			// IPv4:     ip,
			// DERPPort: port,
		}

		node.IPv4 = r.IP

		if strings.HasPrefix(r.Host, "http") {
			u, err := url.Parse(r.Host)
			if err != nil || u.Host == "" {
				zap.L().Debug("invalid host", zap.String("host", r.Host))
				continue
			}
			node.HostName = u.Hostname()
			node.Name = u.String()
		} else {
			node.HostName = r.IP
			node.Name = fmt.Sprintf("%s://%s:%s", r.Protocol, r.IP, r.Port)
		}

		if node.HostName == node.IPv4 {
			node.InsecureForTests = true
		}

		port, err := strconv.Atoi(r.Port)
		if err != nil {
			zap.L().Debug("invalid port", zap.String("port", r.Port))
			continue
		}

		node.DERPPort = port

		regionName := fmt.Sprintf("%s-%s-%s-%s", r.ASOrganization, r.Country, r.Region, node.Name)
		derpMap.Regions[regionID] = &tailcfg.DERPRegion{
			RegionID:   regionID,
			RegionName: regionName,
			RegionCode: regionName,
			Nodes: []*tailcfg.DERPNode{
				node,
			},
		}
	}
	return derpMap, nil
}
