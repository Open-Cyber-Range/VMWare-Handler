/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Tier1 gateway state
type Tier1GatewayState struct {
	// String Path of the enforcement point. When not specified, routes from all enforcement-points are returned. 
	EnforcementPointPath string `json:"enforcement_point_path,omitempty"`
	// IPv6 DAD status for interfaces configured on Tier1 
	Ipv6Status []IPv6Status `json:"ipv6_status,omitempty"`
	Tier1State *LogicalRouterState `json:"tier1_state,omitempty"`
	Tier1Status *LogicalRouterStatus `json:"tier1_status,omitempty"`
	TransportZone *PolicyTransportZone `json:"transport_zone,omitempty"`
}
