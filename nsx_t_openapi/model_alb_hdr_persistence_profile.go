/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HdrPersistenceProfile object
type AlbHdrPersistenceProfile struct {
	// Header name for custom header persistence.
	PrstHdrName string `json:"prst_hdr_name,omitempty"`
}
