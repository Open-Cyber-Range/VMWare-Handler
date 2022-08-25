/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Represents X and Y axis unit of a graph.
type AxisUnit struct {
	// If the condition is met then the above unit will be displayed. to UI. If no condition is provided, then the unit will be displayed unconditionally.
	Condition string `json:"condition,omitempty"`
	// An Axis unit.
	Unit string `json:"unit,omitempty"`
}
