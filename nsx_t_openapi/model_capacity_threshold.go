/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type CapacityThreshold struct {
	// Set the maximum threshold percentage. Specify a value between 0 and 100. Usage percentage above this value is tagged as critical. 
	MaxThresholdPercentage float32 `json:"max_threshold_percentage"`
	// Set the minimum threshold percentage. Specify a value between 0 and 100. Usage percentage above this value is tagged as warning. 
	MinThresholdPercentage float32 `json:"min_threshold_percentage"`
	// Indicate the object type for which threshold is to be set. 
	ThresholdType string `json:"threshold_type"`
}
