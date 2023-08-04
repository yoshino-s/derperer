package derperer

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

var id = 114000
var idMap = map[string]int{}

func CountDERPMap(derpMap *tailcfg.DERPMap) int {
	count := 0
	for _, region := range derpMap.Regions {
		count += len(region.Nodes)
	}
	return count
}

func getId(ip string) int {
	if id, ok := idMap[ip]; ok {
		return id
	}
	id++
	idMap[ip] = id
	return id
}

func Convert(result []fofa.FofaResult) (*tailcfg.DERPMap, error) {
	derpMap := &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}

	for _, r := range result {
		ip := net.ParseIP(r.IP).To4()
		if ip == nil {
			zap.L().Debug("invalid ip", zap.String("ip", r.IP))
			continue
		}

		u, err := url.Parse(r.Host)
		if err != nil || u.Host == "" {
			zap.L().Debug("invalid host", zap.String("host", r.Host))
			continue
		}
		host := u.Hostname()

		port, err := strconv.Atoi(r.Port)
		if err != nil {
			zap.L().Debug("invalid port", zap.String("port", r.Port))
			continue
		}

		nodeName := u.String()

		regionName := fmt.Sprintf("%s-%s-%s", r.ASOrganization, r.Country, r.Region)

		regionID := getId(regionName)

		node := &tailcfg.DERPNode{
			Name:             nodeName,
			RegionID:         regionID,
			HostName:         host,
			IPv4:             r.IP,
			InsecureForTests: true,
			DERPPort:         port,
		}

		if _, ok := derpMap.Regions[regionID]; !ok {
			derpMap.Regions[regionID] = &tailcfg.DERPRegion{
				RegionID:   regionID,
				RegionName: regionName,
				RegionCode: regionName,
				Nodes: []*tailcfg.DERPNode{
					node,
				},
			}
		} else {
			replaced := false
			for idx, prevNode := range derpMap.Regions[regionID].Nodes {
				if prevNode.IPv4 == prevNode.HostName && prevNode.IPv4 == node.IPv4 && prevNode.DERPPort == node.DERPPort {
					// prevNode is a ip, replace
					derpMap.Regions[regionID].Nodes[idx] = node
					replaced = true
				}
			}
			if !replaced {
				derpMap.Regions[regionID].Nodes = append(derpMap.Regions[regionID].Nodes, node)
			}
		}
	}
	return derpMap, nil
}
