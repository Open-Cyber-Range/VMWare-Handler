/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer PGDeploymentRule object
type AlbpgDeploymentRule struct {
	// metric_id of PGDeploymentRule. Default value when not specified in API or module is interpreted by ALB Controller as health.health_score_value. 
	MetricId string `json:"metric_id,omitempty"`
	// Enum options - CO_EQ, CO_GT, CO_GE, CO_LT, CO_LE, CO_NE. Default value when not specified in API or module is interpreted by ALB Controller as CO_GE. 
	Operator string `json:"operator,omitempty"`
	// metric threshold that is used as the pass fail. If it is not provided then it will simply compare it with current pool vs new pool. 
	Threshold float32 `json:"threshold,omitempty"`
}
