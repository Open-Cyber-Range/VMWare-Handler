/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer AnalyticsPolicy object
type AlbAnalyticsPolicy struct {
	// Log all headers. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	AllHeaders bool `json:"all_headers,omitempty"`
	// Gain insights from sampled client to server HTTP requests and responses. Enum options - NO_INSIGHTS, PASSIVE, ACTIVE. Default value when not specified in API or module is interpreted by ALB Controller as NO_INSIGHTS. 
	ClientInsights string `json:"client_insights,omitempty"`
	ClientInsightsSampling *AlbClientInsightsSampling `json:"client_insights_sampling,omitempty"`
	// Placeholder for description of property client_log_filters of obj type AnalyticsPolicy field type str  type array. 
	ClientLogFilters []AlbClientLogFilter `json:"client_log_filters,omitempty"`
	FullClientLogs *AlbFullClientLogs `json:"full_client_logs,omitempty"`
	MetricsRealtimeUpdate *AlbMetricsRealTimeUpdate `json:"metrics_realtime_update,omitempty"`
	// This setting limits the number of significant logs generated per second for this VS on each SE. Default is 10 logs per second. Set it to zero (0) to deactivate throttling. Unit is PER_SECOND. Default value when not specified in API or module is interpreted by ALB Controller as 10. 
	SignificantLogThrottle int64 `json:"significant_log_throttle,omitempty"`
	// This setting limits the total number of UDF logs generated per second for this VS on each SE. UDF logs are generated due to the configured client log filters or the rules with logging enabled. Default is 10 logs per second. Set it to zero (0) to deactivate throttling. Unit is PER_SECOND. Default value when not specified in API or module is interpreted by ALB Controller as 10. 
	UdfLogThrottle int64 `json:"udf_log_throttle,omitempty"`
}
