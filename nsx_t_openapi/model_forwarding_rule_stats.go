/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// FP Rule Statistics. 
type ForwardingRuleStats struct {
	// The server will populate this field when returing the resource. Ignored on PUT and POST.
	Links []ResourceLink `json:"_links,omitempty"`
	// Schema for this resource
	Schema string `json:"_schema,omitempty"`
	Self *SelfResourceLink `json:"_self,omitempty"`
	// Aggregated number of bytes processed by the rule. 
	ByteCount int64 `json:"byte_count,omitempty"`
	// Aggregated number of hits received by the rule.
	HitCount int64 `json:"hit_count,omitempty"`
	// Realized id of the rule on NSX MP. Policy Manager can create more than one rule per policy rule, in which case this identifier helps to distinguish between the multple rules created. 
	InternalRuleId string `json:"internal_rule_id,omitempty"`
	// Aggregated number of L7 Profile Accepted counters received by the rule.
	L7AcceptCount int64 `json:"l7_accept_count,omitempty"`
	// Aggregated number of L7 Profile Rejected counters received by the rule.
	L7RejectCount int64 `json:"l7_reject_count,omitempty"`
	// Aggregated number of L7 Profile Rejected with Response counters received by the rule.
	L7RejectWithResponseCount int64 `json:"l7_reject_with_response_count,omitempty"`
	// Path of the LR on which the section is applied in case of Edge FW.
	LrPath string `json:"lr_path,omitempty"`
	// Maximum value of popularity index of all rules of the type. This is aggregated statistic which are computed with lower frequency compared to individual generic rule statistics. It may have a computation delay up to 15 minutes in response to this API. 
	MaxPopularityIndex int64 `json:"max_popularity_index,omitempty"`
	// Maximum value of sessions count of all rules of the type. This is aggregated statistic which are computed with lower frequency compared to generic rule statistics. It may have a computation delay up to 15 minutes in response to this API. 
	MaxSessionCount int64 `json:"max_session_count,omitempty"`
	// Aggregated number of packets processed by the rule. 
	PacketCount int64 `json:"packet_count,omitempty"`
	// This is calculated by sessions count divided by age of the rule.
	PopularityIndex int64 `json:"popularity_index,omitempty"`
	// Path of the rule.
	Rule string `json:"rule,omitempty"`
	// Aggregated number of sessions processed by the rule. 
	SessionCount int64 `json:"session_count,omitempty"`
	// Aggregated number of sessions processed by all the rules This is aggregated statistic which are computed with lower frequency compared to individual generic rule  statistics. It may have a computation delay up to 15 minutes in response to this API. 
	TotalSessionCount int64 `json:"total_session_count,omitempty"`
}
