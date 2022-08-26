/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced configuration for Policy connectivity
type ConnectivityAdvancedConfig struct {
	// Connectivity configuration to manually connect (ON) or disconnect (OFF) Tier-0/Tier1 segment from corresponding gateway. This property does not apply to VLAN backed segments. VLAN backed segments with connectivity OFF does not affect its layer-2 connectivity. 
	Connectivity string `json:"connectivity,omitempty"`
}
