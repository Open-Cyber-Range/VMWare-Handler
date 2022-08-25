/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Describes usage summary of virtual servers, pools and pool members for all load balancer services. 
type LbServiceUsageSummary struct {
	// The current count of pools configured for all load balancer services. 
	CurrentPoolCount int64 `json:"current_pool_count,omitempty"`
	// The current count of pool members configured for all load balancer services. 
	CurrentPoolMemberCount int64 `json:"current_pool_member_count,omitempty"`
	// The current count of virtual servers configured for all load balancer services. 
	CurrentVirtualServerCount int64 `json:"current_virtual_server_count,omitempty"`
	// Pool capacity means maximum number of pools which can be configured for all load balancer services. 
	PoolCapacity int64 `json:"pool_capacity,omitempty"`
	// Pool capacity means maximum number of pool members which can be configured for all load balancer services. 
	PoolMemberCapacity int64 `json:"pool_member_capacity,omitempty"`
	// The severity calculation is based on the overall usage percentage of pool members for all load balancer services. 
	PoolMemberSeverity string `json:"pool_member_severity,omitempty"`
	// Overall pool member usage percentage for all load balancer services. 
	PoolMemberUsagePercentage float32 `json:"pool_member_usage_percentage,omitempty"`
	// The severity calculation is based on the overall usage percentage of pools for all load balancer services. 
	PoolSeverity string `json:"pool_severity,omitempty"`
	// Overall pool usage percentage for all load balancer services. 
	PoolUsagePercentage float32 `json:"pool_usage_percentage,omitempty"`
	// The service count for each load balancer usage severity. 
	ServiceCounts []LbServiceCountPerSeverity `json:"service_counts,omitempty"`
	// The property identifies all lb service usages. By default, it is not included in response. It exists when parameter ?include_usages=true. 
	ServiceUsages []LbServiceUsage `json:"service_usages,omitempty"`
	// Virtual server capacity means maximum number of virtual servers which can be configured for all load balancer services. 
	VirtualServerCapacity int64 `json:"virtual_server_capacity,omitempty"`
	// The severity calculation is based on the overall usage percentage of virtual servers for all load balancer services. 
	VirtualServerSeverity string `json:"virtual_server_severity,omitempty"`
	// Overall virtual server usage percentage for all load balancer services. 
	VirtualServerUsagePercentage float32 `json:"virtual_server_usage_percentage,omitempty"`
}
