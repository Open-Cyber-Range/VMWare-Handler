/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer PortMatch object
type AlbPortMatch struct {
	// Criterion to use for port matching the HTTP request. Enum options - IS_IN, IS_NOT_IN. 
	MatchCriteria string `json:"match_criteria"`
	// Listening TCP port(s). Allowed values are 1-65535. Minimum of 1 items required. 
	Ports []int64 `json:"ports"`
}