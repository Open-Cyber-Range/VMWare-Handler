/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer FailActionHTTPRedirect object
type AlbFailActionHttpRedirect struct {
	// host of FailActionHTTPRedirect.
	Host string `json:"host"`
	// path of FailActionHTTPRedirect.
	Path string `json:"path,omitempty"`
	// Enum options - HTTP, HTTPS. Allowed in Basic(Allowed values- HTTP) edition, Enterprise edition. Special default for Basic edition is HTTP, Enterprise is HTTPS. Default value when not specified in API or module is interpreted by ALB Controller as HTTP. 
	Protocol string `json:"protocol,omitempty"`
	// query of FailActionHTTPRedirect.
	Query string `json:"query,omitempty"`
	// Enum options - HTTP_REDIRECT_STATUS_CODE_301, HTTP_REDIRECT_STATUS_CODE_302, HTTP_REDIRECT_STATUS_CODE_307. Allowed in Basic(Allowed values- HTTP_REDIRECT_STATUS_CODE_302) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as HTTP_REDIRECT_STATUS_CODE_302. 
	StatusCode string `json:"status_code,omitempty"`
}
