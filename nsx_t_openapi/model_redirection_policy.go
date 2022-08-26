/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Ordered list of rules long with the path of PolicyServiceInstance to which the traffic needs to be redirected. | Please note that the scope property must be provided for NS redirection | policy if redirect to is a service chain. For NS, when redirect to is not | to the service chain, and scope is specified on RedirectionPolicy, it | will be ignored. The scope will be determined from redirect to path | instead. For EW policy, scope must not be  supplied in the request. | Path to either Tier0 or Tier1 is allowed as the scope. Only 1 path | can be specified as a scope. | Also, note that, if stateful flag is not sent, it will be treated as true. If statelessness is intended, false must be sent explicitly as the value | for stateful field. 
type RedirectionPolicy struct {
	// The server will populate this field when returing the resource. Ignored on PUT and POST.
	Links []ResourceLink `json:"_links,omitempty"`
	// Schema for this resource
	Schema string `json:"_schema,omitempty"`
	Self *SelfResourceLink `json:"_self,omitempty"`
	// The _revision property describes the current revision of the resource. To prevent clients from overwriting each other's changes, PUT operations must include the current _revision of the resource, which clients should obtain by issuing a GET operation. If the _revision provided in a PUT request is missing or stale, the operation will be rejected.
	Revision int32 `json:"_revision,omitempty"`
	// Timestamp of resource creation
	CreateTime int64 `json:"_create_time,omitempty"`
	// ID of the user who created this resource
	CreateUser string `json:"_create_user,omitempty"`
	// Timestamp of last modification
	LastModifiedTime int64 `json:"_last_modified_time,omitempty"`
	// ID of the user who last modified this resource
	LastModifiedUser string `json:"_last_modified_user,omitempty"`
	// Protection status is one of the following: PROTECTED - the client who retrieved the entity is not allowed             to modify it. NOT_PROTECTED - the client who retrieved the entity is allowed                 to modify it REQUIRE_OVERRIDE - the client who retrieved the entity is a super                    user and can modify it, but only when providing                    the request header X-Allow-Overwrite=true. UNKNOWN - the _protection field could not be determined for this           entity. 
	Protection string `json:"_protection,omitempty"`
	// Indicates system owned resource
	SystemOwned bool `json:"_system_owned,omitempty"`
	// Description of this resource
	Description string `json:"description,omitempty"`
	// Defaults to ID if not set
	DisplayName string `json:"display_name,omitempty"`
	// Unique identifier of this resource
	Id string `json:"id,omitempty"`
	// The type of this resource.
	ResourceType string `json:"resource_type,omitempty"`
	// Opaque identifiers meaningful to the API user
	Tags []Tag `json:"tags,omitempty"`
	// Path of its parent
	ParentPath string `json:"parent_path,omitempty"`
	// Absolute path of this object
	Path string `json:"path,omitempty"`
	// This is a UUID generated by the system for realizing the entity object. In most cases this should be same as 'unique_id' of the entity. However, in some cases this can be different because of entities have migrated thier unique identifier to NSX Policy intent objects later in the timeline and did not use unique_id for realization. Realization id is helpful for users to debug data path to correlate the configuration with corresponding intent. 
	RealizationId string `json:"realization_id,omitempty"`
	// Path relative from its parent
	RelativePath string `json:"relative_path,omitempty"`
	// This is a UUID generated by the GM/LM to uniquely identify entites in a federated environment. For entities that are stretched across multiple sites, the same ID will be used on all the stretched sites. 
	UniqueId string `json:"unique_id,omitempty"`
	// subtree for this type within policy tree containing nested elements. 
	Children []ChildPolicyConfigResource `json:"children,omitempty"`
	// Intent objects are not directly deleted from the system when a delete is invoked on them. They are marked for deletion and only when all the realized entities for that intent object gets deleted, the intent object is deleted. Objects that are marked for deletion are not returned in GET call. One can use the search API to get these objects. 
	MarkedForDelete bool `json:"marked_for_delete,omitempty"`
	// Global intent objects cannot be modified by the user. However, certain global intent objects can be overridden locally by use of this property. In such cases, the overridden local values take precedence over the globally defined values for the properties. 
	Overridden bool `json:"overridden,omitempty"`
	// - Distributed Firewall - Policy framework provides five pre-defined categories for classifying a security policy. They are \"Ethernet\",\"Emergency\", \"Infrastructure\" \"Environment\" and \"Application\". There is a pre-determined order in which the policy framework manages the priority of these security policies. Ethernet category is for supporting layer 2 firewall rules. The other four categories are applicable for layer 3 rules. Amongst them, the Emergency category has the highest priority followed by Infrastructure, Environment and then Application rules. Administrator can choose to categorize a security policy into the above categories or can choose to leave it empty. If empty it will have the least precedence w.r.t the above four categories. - Edge Firewall - Policy Framework for Edge Firewall provides six pre-defined categories \"Emergency\", \"SystemRules\", \"SharedPreRules\", \"LocalGatewayRules\", \"AutoServiceRules\" and \"Default\", in order of priority of rules. All categories are allowed for Gatetway Policies that belong to 'default' Domain. However, for user created domains, category is restricted to \"SharedPreRules\" or \"LocalGatewayRules\" only. Also, the users can add/modify/delete rules from only the \"SharedPreRules\" and \"LocalGatewayRules\" categories. If user doesn't specify the category then defaulted to \"Rules\". System generated category is used by NSX created rules, for example BFD rules. Autoplumbed category used by NSX verticals to autoplumb data path rules. Finally, \"Default\" category is the placeholder default rules with lowest in the order of priority. 
	Category string `json:"category,omitempty"`
	// Comments for security policy lock/unlock.
	Comments string `json:"comments,omitempty"`
	// This field is to indicate the internal sequence number of a policy with respect to the policies across categories. 
	InternalSequenceNumber int32 `json:"internal_sequence_number,omitempty"`
	// A flag to indicate whether policy is a default policy.
	IsDefault bool `json:"is_default,omitempty"`
	// ID of the user who last modified the lock for the secruity policy. 
	LockModifiedBy string `json:"lock_modified_by,omitempty"`
	// SecurityPolicy locked/unlocked time in epoch milliseconds.
	LockModifiedTime int64 `json:"lock_modified_time,omitempty"`
	// Indicates whether a security policy should be locked. If the security policy is locked by a user, then no other user would be able to modify this security policy. Once the user releases the lock, other users can update this security policy. 
	Locked bool `json:"locked,omitempty"`
	// The count of rules in the policy. 
	RuleCount int32 `json:"rule_count,omitempty"`
	// Provides a mechanism to apply the rules in this policy for a specified time duration. 
	SchedulerPath string `json:"scheduler_path,omitempty"`
	// The list of group paths where the rules in this policy will get applied. This scope will take precedence over rule level scope. Supported only for security and redirection policies. In case of RedirectionPolicy, it is expected only when the policy is NS and redirecting to service chain. 
	Scope []string `json:"scope,omitempty"`
	// This field is used to resolve conflicts between security policies across domains. In order to change the sequence number of a policy one can fire a POST request on the policy entity with a query parameter action=revise The sequence number field will reflect the value of the computed sequence number upon execution of the above mentioned POST request. For scenarios where the administrator is using a template to update several security policies, the only way to set the sequence number is to explicitly specify the sequence number for each security policy. If no sequence number is specified in the payload, a value of 0 is assigned by default. If there are multiple policies with the same sequence number then their order is not deterministic. If a specific order of policies is desired, then one has to specify unique sequence numbers or use the POST request on the policy entity with a query parameter action=revise to let the framework assign a sequence number. The value of sequence number must be between 0 and 999,999. 
	SequenceNumber int32 `json:"sequence_number,omitempty"`
	// Stateful or Stateless nature of security policy is enforced on all rules in this security policy. When it is stateful, the state of the network connects are tracked and a stateful packet inspection is performed. Layer3 security policies can be stateful or stateless. By default, they are stateful. Layer2 security policies can only be stateless. 
	Stateful bool `json:"stateful,omitempty"`
	// Ensures that a 3 way TCP handshake is done before the data packets are sent. tcp_strict=true is supported only for stateful security policies. If the tcp_strict flag is not specified and the security policy is stateful, then tcp_strict will be set to true. 
	TcpStrict bool `json:"tcp_strict,omitempty"`
	// This is the read only flag which will state the direction of this | redirection policy. True denotes that it is NORTH-SOUTH and false | value means it is an EAST-WEST redirection policy. 
	NorthSouth bool `json:"north_south,omitempty"`
	// Paths to which traffic will be redirected to. As of now, only 1 is | supported. Paths allowed are | 1. Policy Service Instance | 2. Service Instance Endpoint | 3. Virtual Endpoint | 4. Policy Service Chain 
	RedirectTo []string `json:"redirect_to,omitempty"`
	// Redirection rules that are a part of this RedirectionPolicy. At max, there can be 1000 rules in a given RedirectPolicy. 
	Rules []RedirectionRule `json:"rules,omitempty"`
}
