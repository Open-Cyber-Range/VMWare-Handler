/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// The status value of one Antrea traceflow. 
type AntreaTraceflowStatus struct {
	// The execution phase of one traceflow. 
	Phase string `json:"phase,omitempty"`
	// The reason for the failure. 
	Reason string `json:"reason,omitempty"`
}
