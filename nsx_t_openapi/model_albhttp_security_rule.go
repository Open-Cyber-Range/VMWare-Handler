/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HTTPSecurityRule object
type AlbhttpSecurityRule struct {
	Action *AlbhttpSecurityAction `json:"action,omitempty"`
	// Enable or disable the rule. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	Enable bool `json:"enable"`
	// Index of the rule.
	Index int64 `json:"index"`
	// Log HTTP request upon rule match.
	Log bool `json:"log,omitempty"`
	Match *AlbMatchTarget `json:"match,omitempty"`
	// Name of the rule.
	Name string `json:"name"`
}
