/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type OspfRoute struct {
	// OSPF area.
	Area string `json:"area,omitempty"`
	// Cost of the route.
	Cost int64 `json:"cost,omitempty"`
	// request counter.
	NextHops []OspfRouteNextHopResult `json:"next_hops,omitempty"`
	// Learned route prefix.
	RoutePrefix string `json:"route_prefix,omitempty"`
	// Type of route.
	RouteType string `json:"route_type,omitempty"`
	// Type of router.
	RouterType string `json:"router_type,omitempty"`
	// Type to cost of the route.
	TypeToCost int64 `json:"type_to_cost,omitempty"`
}