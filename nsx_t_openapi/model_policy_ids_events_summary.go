/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Intrusion event with all the event and signature details, each event contains the signature id, name, severity, first and recent occurence, users and VMs affected and other signature metadata. 
type PolicyIdsEventsSummary struct {
	// Count of workload IPs on which a particular signature was detected.
	AffectedIpCount int64 `json:"affected_ip_count,omitempty"`
	// Count of VMs on which a particular signature was detected.
	AffectedVmCount int64 `json:"affected_vm_count,omitempty"`
	// First occurence of the intrusion, in epoch milliseconds.
	FirstOccurence int64 `json:"first_occurence,omitempty"`
	// IDS event flow data specific to each IDS event. The data includes source ip, source port, destination ip, destination port, and protocol.
	IdsFlowDetails interface{} `json:"ids_flow_details,omitempty"`
	// Flag indicating an ongoing intrusion.
	IsOngoing bool `json:"is_ongoing,omitempty"`
	// Indicates if the rule id is valid or not.
	IsRuleValid bool `json:"is_rule_valid,omitempty"`
	// Latest occurence of the intrusion, in epoch milliseconds.
	LatestOccurence int64 `json:"latest_occurence,omitempty"`
	// IDSEvent resource type.
	ResourceType string `json:"resource_type,omitempty"`
	// The IDS Rule id that detected this particular intrusion.
	RuleId int64 `json:"rule_id,omitempty"`
	// Signature ID pertaining to the detected intrusion.
	SignatureId int64 `json:"signature_id,omitempty"`
	// Metadata about the detected signature including name, id, severity, product affected, protocol etc.
	SignatureMetadata interface{} `json:"signature_metadata,omitempty"`
	// Number of times this particular signature was detected.
	TotalCount int64 `json:"total_count,omitempty"`
	// List of users logged into VMs on which a particular signature was detected.
	UserDetails interface{} `json:"user_details,omitempty"`
	// List of VMs on which a particular signature was detected with the count.
	VmDetails interface{} `json:"vm_details,omitempty"`
}