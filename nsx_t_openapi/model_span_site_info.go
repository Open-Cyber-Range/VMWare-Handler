/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Represents the Site resource information for a Span entity including both the internal id as well as the site path. 
type SpanSiteInfo struct {
	// Site UUID representing the Site resource 
	SiteId string `json:"site_id,omitempty"`
	// Path of the Site resource 
	SitePath string `json:"site_path,omitempty"`
}