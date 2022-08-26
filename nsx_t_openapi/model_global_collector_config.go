/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// The GlobalCollectorConfig is the base class for global collector configurations for different types in a NSX domain. 
type GlobalCollectorConfig struct {
	// IP address for the global collector.
	CollectorIp string `json:"collector_ip"`
	// Port for the global collector.
	CollectorPort int32 `json:"collector_port"`
	// Specify the global collector type.
	CollectorType string `json:"collector_type"`
}
