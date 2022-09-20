/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Duplicate address detection status for IP address on the interface.
type InterfaceIPv6DadStatus struct {
	// Array of edge nodes on which DAD status is reported for given IP address. 
	EdgePaths []string `json:"edge_paths,omitempty"`
	// IP address on the port for which DAD status is reported. 
	IpAddress string `json:"ip_address,omitempty"`
	// DAD status for IP address on the port. 
	Status string `json:"status,omitempty"`
}