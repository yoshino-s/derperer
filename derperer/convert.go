package derperer

import (
	"fmt"
	"net"
	"strconv"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

var id = 114000
var idMap = map[string]int{}

func getId(ip string) int {
	if id, ok := idMap[ip]; ok {
		return id
	}
	id++
	idMap[ip] = id
	return id
}

func Convert(result []fofa.FofaResult) (tailcfg.DERPMap, error) {
	derpMap := tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}

	for _, r := range result {
		ip := net.ParseIP(r.IP).To4()
		if ip == nil {
			zap.L().Debug("invalid ip", zap.String("ip", r.IP))
			continue
		}

		host := r.Domain
		if host == "" {
			host = r.IP
		}

		port, err := strconv.Atoi(r.Port)
		if err != nil {
			zap.L().Debug("invalid port", zap.String("port", r.Port))
			continue
		}

		name := fmt.Sprintf("%s:%d", host, port)

		regionID := getId(fmt.Sprintf("%s:%d", r.IP, port))

		node := &tailcfg.DERPNode{
			Name:             name,
			RegionID:         regionID,
			HostName:         host,
			IPv4:             r.IP,
			InsecureForTests: true,
			DERPPort:         port,
		}

		if _, ok := derpMap.Regions[regionID]; !ok {
			derpMap.Regions[regionID] = &tailcfg.DERPRegion{
				RegionID:   regionID,
				RegionName: r.IP,
				Nodes: []*tailcfg.DERPNode{
					node,
				},
			}
		} else {
			prevNode := derpMap.Regions[regionID].Nodes[0]
			if prevNode.IPv4 == prevNode.HostName {
				// prevNode is a ip, replace
				derpMap.Regions[regionID].Nodes[0] = node
			} else {
				zap.L().Debug("duplicate region", zap.String("region", name), zap.String("prev", prevNode.Name))
			}
		}
	}
	return derpMap, nil
}
