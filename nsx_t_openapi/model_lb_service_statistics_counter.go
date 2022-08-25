/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type LbServiceStatisticsCounter struct {
	// The average number of l4 current sessions per second, the number is averaged over the last 5 one-second intervals. 
	L4CurrentSessionRate float32 `json:"l4_current_session_rate,omitempty"`
	// Number of l4 current sessions.
	L4CurrentSessions int64 `json:"l4_current_sessions,omitempty"`
	// L4 max sessions is used to show the peak L4 max session data since load balancer starts to provide service. 
	L4MaxSessions int64 `json:"l4_max_sessions,omitempty"`
	// Number of l4 total sessions.
	L4TotalSessions int64 `json:"l4_total_sessions,omitempty"`
	// The average number of l7 current requests per second, the number is averaged over the last 5 one-second intervals. 
	L7CurrentSessionRate float32 `json:"l7_current_session_rate,omitempty"`
	// Number of l7 current sessions.
	L7CurrentSessions int64 `json:"l7_current_sessions,omitempty"`
	// L7 max sessions is used to show the peak L7 max session data since load balancer starts to provide service. 
	L7MaxSessions int64 `json:"l7_max_sessions,omitempty"`
	// Number of l7 total sessions.
	L7TotalSessions int64 `json:"l7_total_sessions,omitempty"`
}
