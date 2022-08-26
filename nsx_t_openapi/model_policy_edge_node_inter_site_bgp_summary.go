/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type PolicyEdgeNodeInterSiteBgpSummary struct {
	// Edge node path whose status is being reported.
	EdgeNodePath string `json:"edge_node_path,omitempty"`
	// Timestamp when the inter-site IBGP neighbors status was last updated. 
	LastUpdateTimestamp int64 `json:"last_update_timestamp,omitempty"`
	// Status of all inter-site IBGP neighbors.
	NeighborStatus []PolicyBgpNeighborStatus `json:"neighbor_status,omitempty"`
}
