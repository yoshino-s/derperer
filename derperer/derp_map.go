package derperer

import (
	"sort"

	"tailscale.com/tailcfg"
)

type DERPRegionStatus int8

const (
	DERPRegionStatusUnknown DERPRegionStatus = iota
	DERPRegionStatusAlive
	DERPRegionStatusError
)

// DERPMap describes the set of DERP packet relay servers that are available.
type DERPMap struct {
	// HomeParams, if non-nil, is a change in home parameters.
	//
	// The rest of the DEPRMap fields, if zero, means unchanged.
	HomeParams *DERPHomeParams `json:",omitempty"`

	// Regions is the set of geographic regions running DERP node(s).
	//
	// It's keyed by the DERPRegion.RegionID.
	//
	// The numbers are not necessarily contiguous.
	Regions map[int]*DERPRegion

	// OmitDefaultRegions specifies to not use Tailscale's DERP servers, and only use those
	// specified in this DERPMap. If there are none set outside of the defaults, this is a noop.
	//
	// This field is only meaningful if the Regions map is non-nil (indicating a change).
	OmitDefaultRegions bool `json:"omitDefaultRegions,omitempty"`
}

func NewDERPMap() *DERPMap {
	return &DERPMap{
		Regions: map[int]*DERPRegion{},
		HomeParams: &DERPHomeParams{
			RegionScore: map[int]float64{},
		},
	}
}

func (src *DERPMap) Clone() *DERPMap {
	if src == nil {
		return nil
	}
	dst := new(DERPMap)
	*dst = *src
	dst.HomeParams = src.HomeParams.Clone()
	if dst.Regions != nil {
		dst.Regions = map[int]*DERPRegion{}
		for k, v := range src.Regions {
			dst.Regions[k] = v.Clone()
		}
	}
	return dst
}

func (m *DERPMap) Convert() *tailcfg.DERPMap {
	ret := &tailcfg.DERPMap{
		HomeParams:         m.HomeParams.Convert(),
		OmitDefaultRegions: m.OmitDefaultRegions,
	}
	for idx, r := range m.Regions {
		ret.Regions[idx] = r.Convert()
	}
	return ret
}

// / RegionIDs returns the sorted region IDs.
func (m *DERPMap) RegionIDs() []int {
	ret := make([]int, 0, len(m.Regions))
	for rid := range m.Regions {
		ret = append(ret, rid)
	}
	sort.Ints(ret)
	return ret
}

// DERPHomeParams contains parameters from the server related to selecting a
// DERP home region (sometimes referred to as the "preferred DERP").
type DERPHomeParams struct {
	// RegionScore scales latencies of DERP regions by a given scaling
	// factor when determining which region to use as the home
	// ("preferred") DERP. Scores in the range (0, 1) will cause this
	// region to be proportionally more preferred, and scores in the range
	// (1, ∞) will penalize a region.
	//
	// If a region is not present in this map, it is treated as having a
	// score of 1.0.
	//
	// Scores should not be 0 or negative; such scores will be ignored.
	//
	// A nil map means no change from the previous value (if any); an empty
	// non-nil map can be sent to reset all scores back to 1.0.
	RegionScore map[int]float64 `json:",omitempty"`
}

func (src *DERPHomeParams) Clone() *DERPHomeParams {
	if src == nil {
		return nil
	}
	dst := new(DERPHomeParams)
	*dst = *src
	if dst.RegionScore != nil {
		dst.RegionScore = map[int]float64{}
		for k, v := range src.RegionScore {
			dst.RegionScore[k] = v
		}
	}
	return dst
}

func (p *DERPHomeParams) Convert() *tailcfg.DERPHomeParams {
	ret := &tailcfg.DERPHomeParams{}
	if p == nil {
		return ret
	}
	ret.RegionScore = p.RegionScore
	return ret
}

// DERPRegion is a geographic region running DERP relay node(s).
//
// Client nodes discover which region they're closest to, advertise
// that "home" DERP region (previously called "home node", when there
// was only 1 node per region) and maintain a persistent connection
// that region as long as it's the closest. Client nodes will further
// connect to other regions as necessary to communicate with peers
// advertising other regions as their homes.
type DERPRegion struct {
	// RegionID is a unique integer for a geographic region.
	//
	// It corresponds to the legacy derpN.tailscale.com hostnames
	// used by older clients. (Older clients will continue to resolve
	// derpN.tailscale.com when contacting peers, rather than use
	// the server-provided DERPMap)
	//
	// RegionIDs must be non-zero, positive, and guaranteed to fit
	// in a JavaScript number.
	//
	// RegionIDs in range 900-999 are reserved for end users to run their
	// own DERP nodes.
	RegionID int

	// RegionCode is a short name for the region. It's usually a popular
	// city or airport code in the region: "nyc", "sf", "sin",
	// "fra", etc.
	RegionCode string

	// RegionName is a long English name for the region: "New York City",
	// "San Francisco", "Singapore", "Frankfurt", etc.
	RegionName string

	// Avoid is whether the client should avoid picking this as its home
	// region. The region should only be used if a peer is there.
	// Clients already using this region as their home should migrate
	// away to a new region without Avoid set.
	Avoid bool `json:",omitempty"`

	// Nodes are the DERP nodes running in this region, in
	// priority order for the current client. Client TLS
	// connections should ideally only go to the first entry
	// (falling back to the second if necessary). STUN packets
	// should go to the first 1 or 2.
	//
	// If nodes within a region route packets amongst themselves,
	// but not to other regions. That said, each user/domain
	// should get a the same preferred node order, so if all nodes
	// for a user/network pick the first one (as they should, when
	// things are healthy), the inter-cluster routing is minimal
	// to zero.
	Nodes []*DERPNode

	Bandwidth string `json:",omitempty"`

	Latency string `json:",omitempty"`

	Error string `json:",omitempty"`

	Status DERPRegionStatus `json:""`
}

func (src *DERPRegion) Clone() *DERPRegion {
	if src == nil {
		return nil
	}
	dst := new(DERPRegion)
	*dst = *src
	dst.Nodes = make([]*DERPNode, len(src.Nodes))
	for i := range dst.Nodes {
		dst.Nodes[i] = src.Nodes[i].Clone()
	}
	return dst
}

func (r *DERPRegion) Convert() *tailcfg.DERPRegion {
	ret := &tailcfg.DERPRegion{
		RegionID:   r.RegionID,
		RegionCode: r.RegionCode,
		RegionName: r.RegionName,
		Avoid:      r.Avoid,
	}
	for _, n := range r.Nodes {
		ret.Nodes = append(ret.Nodes, n.Convert())
	}
	return ret
}

// DERPNode describes a DERP packet relay node running within a DERPRegion.
type DERPNode struct {
	// Name is a unique node name (across all regions).
	// It is not a host name.
	// It's typically of the form "1b", "2a", "3b", etc. (region
	// ID + suffix within that region)
	Name string

	// RegionID is the RegionID of the DERPRegion that this node
	// is running in.
	RegionID int

	// HostName is the DERP node's hostname.
	//
	// It is required but need not be unique; multiple nodes may
	// have the same HostName but vary in configuration otherwise.
	HostName string

	// CertName optionally specifies the expected TLS cert common
	// name. If empty, HostName is used. If CertName is non-empty,
	// HostName is only used for the TCP dial (if IPv4/IPv6 are
	// not present) + TLS ClientHello.
	CertName string `json:",omitempty"`

	// IPv4 optionally forces an IPv4 address to use, instead of using DNS.
	// If empty, A record(s) from DNS lookups of HostName are used.
	// If the string is not an IPv4 address, IPv4 is not used; the
	// conventional string to disable IPv4 (and not use DNS) is
	// "none".
	IPv4 string `json:",omitempty"`

	// IPv6 optionally forces an IPv6 address to use, instead of using DNS.
	// If empty, AAAA record(s) from DNS lookups of HostName are used.
	// If the string is not an IPv6 address, IPv6 is not used; the
	// conventional string to disable IPv6 (and not use DNS) is
	// "none".
	IPv6 string `json:",omitempty"`

	// Port optionally specifies a STUN port to use.
	// Zero means 3478.
	// To disable STUN on this node, use -1.
	STUNPort int `json:",omitempty"`

	// STUNOnly marks a node as only a STUN server and not a DERP
	// server.
	STUNOnly bool `json:",omitempty"`

	// DERPPort optionally provides an alternate TLS port number
	// for the DERP HTTPS server.
	//
	// If zero, 443 is used.
	DERPPort int `json:",omitempty"`

	// InsecureForTests is used by unit tests to disable TLS verification.
	// It should not be set by users.
	InsecureForTests bool `json:",omitempty"`

	// STUNTestIP is used in tests to override the STUN server's IP.
	// If empty, it's assumed to be the same as the DERP server.
	STUNTestIP string `json:",omitempty"`

	// CanPort80 specifies whether this DERP node is accessible over HTTP
	// on port 80 specifically. This is used for captive portal checks.
	CanPort80 bool `json:",omitempty"`
}

func (src *DERPNode) Clone() *DERPNode {
	if src == nil {
		return nil
	}
	dst := new(DERPNode)
	*dst = *src
	return dst
}

func (n *DERPNode) Convert() *tailcfg.DERPNode {
	return &tailcfg.DERPNode{
		Name:             n.Name,
		RegionID:         n.RegionID,
		HostName:         n.HostName,
		CertName:         n.CertName,
		IPv4:             n.IPv4,
		IPv6:             n.IPv6,
		STUNPort:         n.STUNPort,
		STUNOnly:         n.STUNOnly,
		DERPPort:         n.DERPPort,
		InsecureForTests: n.InsecureForTests,
		STUNTestIP:       n.STUNTestIP,
		CanPort80:        n.CanPort80,
	}
}

// DotInvalid is a fake DNS TLD used in tests for an invalid hostname.
const DotInvalid = ".invalid"
