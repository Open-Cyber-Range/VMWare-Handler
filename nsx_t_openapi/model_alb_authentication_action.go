/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer AuthenticationAction object
type AlbAuthenticationAction struct {
	// Authentication Action to be taken for a matched Rule. Enum options - SKIP_AUTHENTICATION, USE_DEFAULT_AUTHENTICATION. Default value when not specified in API or module is interpreted by ALB Controller as USE_DEFAULT_AUTHENTICATION. 
	Type_ string `json:"type,omitempty"`
}
