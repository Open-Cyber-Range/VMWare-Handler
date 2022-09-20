/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HTTPVersionMatch object
type AlbhttpVersionMatch struct {
	// Criterion to use for HTTP version matching the version used in the HTTP request. Enum options - IS_IN, IS_NOT_IN. 
	MatchCriteria string `json:"match_criteria"`
	// HTTP protocol version. Enum options - ZERO_NINE, ONE_ZERO, ONE_ONE, TWO_ZERO. Minimum of 1 items required. Maximum of 8 items allowed. Allowed in Basic(Allowed values- ONE_ZERO,ONE_ONE) edition, Essentials(Allowed values- ONE_ZERO,ONE_ONE) edition, Enterprise edition. 
	Versions []string `json:"versions"`
}