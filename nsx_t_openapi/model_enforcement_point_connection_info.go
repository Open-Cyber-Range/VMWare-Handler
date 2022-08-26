/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Contains information required to connect to enforcement point.
type EnforcementPointConnectionInfo struct {
	// Value of this property could be Hostname or IP. For instance: - On an NSX-T MP running on default port, the value could be \"10.192.1.1\" - On an NSX-T MP running on custom port, the value could be \"192.168.1.1:32789\" - On an NSX-T MP in VMC deployments, the value could be \"192.168.1.1:5480/nsxapi\" 
	EnforcementPointAddress string `json:"enforcement_point_address"`
	// Resource Type of Enforcement Point Connection Info.
	ResourceType string `json:"resource_type"`
}
