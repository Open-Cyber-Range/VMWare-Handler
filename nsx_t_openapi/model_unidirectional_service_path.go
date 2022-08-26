/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Representing either forward or reverse service path for ingress or egress traffic respectively.
type UnidirectionalServicePath struct {
	// List of service path hops that constitutes the forward or reverse service path.
	Hops []ServicePathHop `json:"hops,omitempty"`
	// The number of times the traffic needs to cross hosts for the given forward or reverse service path.
	HostCrossCount int64 `json:"host_cross_count,omitempty"`
	// Is forward or revserse service path in maintenance mode or not.
	InMaintenanceMode bool `json:"in_maintenance_mode,omitempty"`
	// Is forward or revserse service path active or not.
	IsActive bool `json:"is_active,omitempty"`
	// Unique identifier of one directional service path.
	UnidirServicePathId int64 `json:"unidir_service_path_id,omitempty"`
}
