/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Standard host switch specification is used for NSX configured transport node.
type StandardHostSwitchSpec struct {
	ResourceType string `json:"resource_type"`
	// Transport Node host switches
	HostSwitches []StandardHostSwitch `json:"host_switches"`
}