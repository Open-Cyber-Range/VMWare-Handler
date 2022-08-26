/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// A set of operations to be performed in a single batch
type BatchRequest struct {
	// Continue even if an error is encountered.
	ContinueOnError bool `json:"continue_on_error,omitempty"`
	Requests []BatchRequestItem `json:"requests,omitempty"`
}
