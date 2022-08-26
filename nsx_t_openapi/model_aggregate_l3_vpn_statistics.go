/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Aggregate of L3Vpn Statistics across Enforcement Points. 
type AggregateL3VpnStatistics struct {
	// Intent path of object, forward slashes must be escaped using %2F. 
	IntentPath string `json:"intent_path"`
	// List of L3Vpn Statistics per Enforcement Point. 
	L3vpnStatisticsPerEnforcementPoint []L3VpnStatisticsPerEnforcementPoint `json:"l3vpn_statistics_per_enforcement_point,omitempty"`
}
