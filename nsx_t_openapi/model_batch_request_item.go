/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// A single request within a batch of operations
type BatchRequestItem struct {
	Body interface{} `json:"body,omitempty"`
	// http method type
	Method string `json:"method"`
	// relative uri (path and args), of the call including resource id (if this is a POST/DELETE), exclude hostname and port and prefix, exploded form of parameters
	Uri string `json:"uri"`
}