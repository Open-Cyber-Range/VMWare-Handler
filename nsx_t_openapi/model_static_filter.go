/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Static filters
type StaticFilter struct {
	// An additional key-value pair for static filter.
	AdditionalValue interface{} `json:"additional_value,omitempty"`
	// display name to be shown in the drop down for static filter.
	DisplayName string `json:"display_name,omitempty"`
	// Property value is shown in the drop down input box for a filter. If the value is not provided 'display_name' property value is used.
	ShortDisplayName string `json:"short_display_name,omitempty"`
	// Value of static filter inside dropdown filter.
	Value string `json:"value,omitempty"`
}