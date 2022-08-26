/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Software module details
type SoftwareModule struct {
	// Name of the module in the node
	ModuleName string `json:"module_name"`
	// Version of the module in the node
	ModuleVersion string `json:"module_version"`
}
