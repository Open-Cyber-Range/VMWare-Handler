/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Routing table. 
type RoutingTable struct {
	// Entry count.
	Count int32 `json:"count,omitempty"`
	// Transport node ID. 
	EdgeNode string `json:"edge_node,omitempty"`
	// Routing table fetch error message, populated only if status if failure. 
	ErrorMessage string `json:"error_message,omitempty"`
	// Route entries.
	RouteEntries []RoutingEntry `json:"route_entries"`
	// Routing table fetch status from Transport node. 
	Status string `json:"status,omitempty"`
}