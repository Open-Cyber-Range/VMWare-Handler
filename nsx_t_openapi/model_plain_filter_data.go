/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type PlainFilterData struct {
	// Filter type
	ResourceType string `json:"resource_type"`
	// Basic RCF rule for packet filter
	BasicFilter string `json:"basic_filter,omitempty"`
	// Extended RCF rule for packet filter
	ExtendFilter string `json:"extend_filter,omitempty"`
}
