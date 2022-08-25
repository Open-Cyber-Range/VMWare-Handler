/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// L2VPN session statistics gives session status and traffic statistics per segment. 
type L2VpnSessionStatisticsNsxT struct {
	Alarm *PolicyRuntimeAlarm `json:"alarm,omitempty"`
	// Policy Path referencing the enforcement point where the info is fetched. 
	EnforcementPointPath string `json:"enforcement_point_path,omitempty"`
	ResourceType string `json:"resource_type"`
	// Display name of l2vpn session.
	DisplayName string `json:"display_name,omitempty"`
	// Tunnel port traffic counters.
	TapTrafficCounters []L2VpnTapStatistics `json:"tap_traffic_counters,omitempty"`
	// Traffic statistics per segment.
	TrafficStatisticsPerSegment []L2VpnTrafficStatisticsPerSegment `json:"traffic_statistics_per_segment,omitempty"`
}
