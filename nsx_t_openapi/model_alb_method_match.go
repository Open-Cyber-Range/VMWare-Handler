/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer MethodMatch object
type AlbMethodMatch struct {
	// Criterion to use for HTTP method matching the method in the HTTP request. Enum options - IS_IN, IS_NOT_IN. 
	MatchCriteria string `json:"match_criteria"`
	// Configure HTTP method(s). Enum options - HTTP_METHOD_GET, HTTP_METHOD_HEAD, HTTP_METHOD_PUT, HTTP_METHOD_DELETE, HTTP_METHOD_POST, HTTP_METHOD_OPTIONS, HTTP_METHOD_TRACE, HTTP_METHOD_CONNECT, HTTP_METHOD_PATCH, HTTP_METHOD_PROPFIND, HTTP_METHOD_PROPPATCH, HTTP_METHOD_MKCOL, HTTP_METHOD_COPY, HTTP_METHOD_MOVE, HTTP_METHOD_LOCK, HTTP_METHOD_UNLOCK. Minimum of 1 items required. Maximum of 16 items allowed. Allowed in Basic(Allowed values- HTTP_METHOD_GET,HTTP_METHOD_PUT,HTTP_METHOD_POST,HTTP_METHOD_HEAD,HTTP_METHOD_OPTIONS) edition, Essentials(Allowed values- HTTP_METHOD_GET,HTTP_METHOD_PUT,HTTP_METHOD_POST,HTTP_METHOD_HEAD,HTTP_METHOD_OPTIONS) edition, Enterprise edition. 
	Methods []string `json:"methods"`
}