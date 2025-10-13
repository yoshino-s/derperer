package derperer

import (
	"tailscale.com/tailcfg"
)

type DERPMap struct {
	tailcfg.DERPMap
	Regions map[int]*DERPRegion
}

type DERPRegion struct {
	tailcfg.DERPRegion
	Nodes []*DERPNode
}

type DERPNode struct {
	tailcfg.DERPNode
	Latency   string     `json:"latency,omitempty"`
	Bandwidth string     `json:"bandwidth,omitempty"`
	Status    DerpStatus `json:"status,omitempty"`
}

func (n *DERPNode) ToOriginal() *tailcfg.DERPNode {
	return &n.DERPNode
}

func (r *DERPRegion) ToOriginal() *tailcfg.DERPRegion {
	region := &r.DERPRegion
	region.Nodes = make([]*tailcfg.DERPNode, 0, len(r.Nodes))
	for _, node := range r.Nodes {
		region.Nodes = append(region.Nodes, node.ToOriginal())
	}
	return region
}

func (m *DERPMap) ToOriginal() *tailcfg.DERPMap {
	derpMap := &m.DERPMap
	derpMap.Regions = make(map[int]*tailcfg.DERPRegion)
	for id, region := range m.Regions {
		derpMap.Regions[id] = region.ToOriginal()
	}
	return derpMap
}
