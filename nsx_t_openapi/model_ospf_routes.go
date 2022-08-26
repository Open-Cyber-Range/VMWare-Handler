/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// OSPF Routes Per Edge. 
type OspfRoutes struct {
	// Display name to edge node. 
	EdgeDisplayName string `json:"edge_display_name,omitempty"`
	// Policy path to edge node. 
	EdgePath string `json:"edge_path"`
	RouteDetails []OspfRoute `json:"route_details,omitempty"`
}
