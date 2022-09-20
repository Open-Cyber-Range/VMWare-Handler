/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Intrusion Detection System Signature . 
type IdsSignature struct {
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
	// Signature action. 
	Action string `json:"action,omitempty"`
	// Target of the signature. 
	AttackTarget string `json:"attack_target,omitempty"`
	// Represents the internal categories a signature belongs to. 
	Categories []string `json:"categories,omitempty"`
	// Class type of Signature. 
	ClassType string `json:"class_type,omitempty"`
	// Signature's confidence score.
	Confidence string `json:"confidence,omitempty"`
	// CVE score 
	Cves []string `json:"cves,omitempty"`
	// Represents the cvss value of a Signature. The value is derived from cvssv3 or cvssv2 score. NONE     means cvssv3/cvssv2 score as 0.0 LOW      means cvssv3/cvssv2 score as 0.1-3.9 MEDIUM   means cvssv3/cvssv2 score as 4.0-6.9 HIGH     means cvssv3/cvssv2 score as 7.0-8.9 CRITICAL means cvssv3/cvssv2 score as 9.0-10.0 
	Cvss string `json:"cvss,omitempty"`
	// Represents the cvss value of a Signature. The value is derived from cvssv3 or cvssv2 score. If cvssv3 exists, then this is the cvssv3 score, else it is the cvssv2 score. 
	CvssScore string `json:"cvss_score,omitempty"`
	// Signature cvssv2 score. 
	Cvssv2 string `json:"cvssv2,omitempty"`
	// Signature cvssv3 score. 
	Cvssv3 string `json:"cvssv3,omitempty"`
	// Source-destination direction.
	Direction string `json:"direction,omitempty"`
	// Flag which tells whether the signature is enabled or not. 
	Enable bool `json:"enable,omitempty"`
	// Flow established from server, from client etc. 
	Flow string `json:"flow,omitempty"`
	// Impact of Signature.
	Impact string `json:"impact,omitempty"`
	// Family of the malware tracked in the signature.
	MalwareFamily string `json:"malware_family,omitempty"`
	// Mitre Attack details of Signature.
	MitreAttack []MitreAttack `json:"mitre_attack,omitempty"`
	// Signature name. 
	Name string `json:"name,omitempty"`
	// Performance impact of the signature.
	PerformanceImpact string `json:"performance_impact,omitempty"`
	// Signature policy.
	Policy []string `json:"policy,omitempty"`
	// Product affected by this signature. 
	ProductAffected string `json:"product_affected,omitempty"`
	// Protocol used in the packet analysis.
	Protocol string `json:"protocol,omitempty"`
	// Risk score of signature.
	RiskScore string `json:"risk_score,omitempty"`
	// Represents the severity of the Signature. 
	Severity string `json:"severity,omitempty"`
	// Represents the Signature's id. 
	SignatureId string `json:"signature_id,omitempty"`
	// Represents revision of the Signature. 
	SignatureRevision string `json:"signature_revision,omitempty"`
	// Signature vendor set severity of the signature rule.
	SignatureSeverity string `json:"signature_severity,omitempty"`
	// Vendor assigned classification tag.
	Tag []string `json:"tag,omitempty"`
	// Signature type.
	Type_ []string `json:"type,omitempty"`
	// List of mitre attack URLs pertaining to signature 
	Urls []string `json:"urls,omitempty"`
}