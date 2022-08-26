/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Aggregate of IPSec VPN Session Statistics across Enforcement Points. 
type AggregateIpSecVpnSessionStatistics struct {
	// Intent path of object, forward slashes must be escaped using %2F. 
	IntentPath string `json:"intent_path,omitempty"`
	// List of IPSec VPN Session Statistics per Enforcement Point. 
	Results []IpSecVpnSessionStatisticsPerEp `json:"results,omitempty"`
}
