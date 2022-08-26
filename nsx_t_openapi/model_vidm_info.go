/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Vidm Info
type VidmInfo struct {
	// User's Full Name Or User Group's Display Name
	DisplayName string `json:"display_name,omitempty"`
	// Username Or Groupname
	Name string `json:"name,omitempty"`
	// Type
	Type_ string `json:"type,omitempty"`
}
