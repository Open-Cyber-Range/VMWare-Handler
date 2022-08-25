/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Multicast PIM configuration.
type Tier0InterfacePimConfig struct {
	// enable/disable PIM configuration. 
	Enabled bool `json:"enabled,omitempty"`
	// PIM hello interval(seconds) at interface level. 
	HelloInterval int32 `json:"hello_interval,omitempty"`
	// PIM hold interval(seconds) at interface level. 
	HoldInterval int32 `json:"hold_interval,omitempty"`
}
