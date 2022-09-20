/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Subnet specification for interface connectivity
type InterfaceSubnet struct {
	// IP addresses assigned to interface
	IpAddresses []string `json:"ip_addresses"`
	// Subnet prefix length
	PrefixLen int32 `json:"prefix_len"`
}