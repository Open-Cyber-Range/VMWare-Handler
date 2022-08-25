/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer ProtocolMatch object
type AlbProtocolMatch struct {
	// Criterion to use for protocol matching the HTTP request. Enum options - IS_IN, IS_NOT_IN. 
	MatchCriteria string `json:"match_criteria"`
	// HTTP or HTTPS protocol. Enum options - HTTP, HTTPS. 
	Protocols string `json:"protocols"`
}
