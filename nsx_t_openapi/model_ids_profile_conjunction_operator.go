/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Represents the operator AND.
type IdsProfileConjunctionOperator struct {
	ResourceType string `json:"resource_type"`
	// IDS Profile Filter Condition
	Operator string `json:"operator"`
}
