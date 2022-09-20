/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Routing table entry. 
type RoutingEntry struct {
	// Admin distance. 
	AdminDistance int32 `json:"admin_distance,omitempty"`
	// The policy path of the interface which is used as the next hop
	Interface_ string `json:"interface,omitempty"`
	// Logical router component(Service Router/Distributed Router) id
	LrComponentId string `json:"lr_component_id,omitempty"`
	// Logical router component(Service Router/Distributed Router) type
	LrComponentType string `json:"lr_component_type,omitempty"`
	// Network CIDR. 
	Network string `json:"network,omitempty"`
	// Next hop address. 
	NextHop string `json:"next_hop,omitempty"`
	// Route type in routing table. t0c - Tier-0 Connected t0s - Tier-0 Static b - BGP t0n - Tier-0 NAT t1s - Tier-1 Static t1c - Tier-1 Connected t1n: Tier-1 NAT t1l: Tier-1 LB VIP t1ls: Tier-1 LB SNAT t1d: Tier-1 DNS FORWARDER t1ipsec: Tier-1 IPSec isr: Inter-SR 
	RouteType string `json:"route_type,omitempty"`
}