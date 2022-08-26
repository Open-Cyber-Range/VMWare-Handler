/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Represents an instance of partner Service and its configuration. 
type PolicyServiceInstance struct {
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
	// Deployment mode specifies how the partner appliance will be deployed i.e. in HA or standalone mode.
	DeploymentMode string `json:"deployment_mode,omitempty"`
	// Unique name of Partner Service in the Marketplace
	PartnerServiceName string `json:"partner_service_name"`
	// Transport to be used while deploying Service-VM.
	TransportType string `json:"transport_type,omitempty"`
	// List of attributes specific to a partner for which the service is created. There attributes are passed on to the partner appliance.
	Attributes []Attribute `json:"attributes"`
	// Id of the compute(ResourcePool) to which this service needs to be deployed.
	ComputeId string `json:"compute_id"`
	// UUID of VCenter/Compute Manager as seen on NSX Manager, to which this service needs to be deployed.
	ContextId string `json:"context_id,omitempty"`
	// Form factor for the deployment of partner service.
	DeploymentSpecName string `json:"deployment_spec_name"`
	// Template for the deployment of partnet service.
	DeploymentTemplateName string `json:"deployment_template_name"`
	// Failure policy for the Service VM. If this values is not provided, it will be defaulted to FAIL_CLOSE.
	FailurePolicy string `json:"failure_policy,omitempty"`
	// Gateway address for primary management console. If the provided segment already has gateway, this field can be omitted. But if it is provided, it takes precedence always. However, if provided segment does not have gateway, this field must be provided. 
	PrimaryGatewayAddress string `json:"primary_gateway_address,omitempty"`
	// Management IP Address of primary interface of the Service
	PrimaryInterfaceMgmtIp string `json:"primary_interface_mgmt_ip"`
	// Path of the segment to which primary interface of the Service VM needs to be connected
	PrimaryInterfaceNetwork string `json:"primary_interface_network,omitempty"`
	// Id of the standard or ditsributed port group for primary management console. Please note that only 1 of the 2 values from 1. primary_interface_network 2. primary_portgroup_id are allowed to be passed. Both can't be passed in the same request. 
	PrimaryPortgroupId string `json:"primary_portgroup_id,omitempty"`
	// Subnet for primary management console IP. If the provided segment already has subnet, this field can be omitted. But if it is provided, it takes precedence always. However, if provided segment does not have subnet, this field must be provided. 
	PrimarySubnetMask string `json:"primary_subnet_mask,omitempty"`
	// Gateway address for secondary management console. If the provided segment already has gateway, this field can be omitted. But if it is provided, it takes precedence always. However, if provided segment does not have gateway, this field must be provided. 
	SecondaryGatewayAddress string `json:"secondary_gateway_address,omitempty"`
	// Management IP Address of secondary interface of the Service
	SecondaryInterfaceMgmtIp string `json:"secondary_interface_mgmt_ip,omitempty"`
	// Path of segment to which secondary interface of the Service VM needs to be connected
	SecondaryInterfaceNetwork string `json:"secondary_interface_network,omitempty"`
	// Id of the standard or ditsributed port group for secondary management console. Please note that only 1 of the 2 values from 1. secondary_interface_network 2. secondary_portgroup_id are allowed to be passed. Both can't be passed in the same request. 
	SecondaryPortgroupId string `json:"secondary_portgroup_id,omitempty"`
	// Subnet for secondary management console IP. If the provided segment already has subnet, this field can be omitted. But if it is provided, it takes precedence always. However, if provided segment does not have subnet, this field must be provided. 
	SecondarySubnetMask string `json:"secondary_subnet_mask,omitempty"`
	// Id of the storage(Datastore). VC moref of Datastore to which this service needs to be deployed.
	StorageId string `json:"storage_id"`
}
