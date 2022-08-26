/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Defining access of a Group from a LBVirtualServer and binding to LBMonitorProfile. 
type LbPool struct {
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
	// In case of active healthchecks, load balancer itself initiates new connections (or sends ICMP ping) to the servers periodically to check their health, completely independent of any data traffic. Active healthchecks are disabled by default and can be enabled for a server pool by binding a health monitor to the pool. If multiple active monitors are configured, the pool member status is UP only when the health check status for all the monitors are UP. The property is deprecated as NSX-T Load Balancer is deprecated. 
	ActiveMonitorPaths []string `json:"active_monitor_paths,omitempty"`
	// Load Balancing algorithm chooses a server for each new connection by going through the list of servers in the pool. Currently, following load balancing algorithms are supported with ROUND_ROBIN as the default. ROUND_ROBIN means that a server is selected in a round-robin fashion. The weight would be ignored even if it is configured. WEIGHTED_ROUND_ROBIN means that a server is selected in a weighted round-robin fashion. Default weight of 1 is used if weight is not configured. LEAST_CONNECTION means that a server is selected when it has the least number of connections. The weight would be ignored even if it is configured. Slow start would be enabled by default. WEIGHTED_LEAST_CONNECTION means that a server is selected in a weighted least connection fashion. Default weight of 1 is used if weight is not configured. Slow start would be enabled by default. IP_HASH means that consistent hash is performed on the source IP address of the incoming connection. This ensures that the same client IP address will always reach the same server as long as no server goes down or up. It may be used on the Internet to provide a best-effort stickiness to clients which refuse session cookies. 
	Algorithm string `json:"algorithm,omitempty"`
	MemberGroup *LbPoolMemberGroup `json:"member_group,omitempty"`
	// Server pool consists of one or more pool members. Each pool member is identified, typically, by an IP address and a port. 
	Members []LbPoolMember `json:"members,omitempty"`
	// A pool is considered active if there are at least certain minimum number of members. 
	MinActiveMembers int64 `json:"min_active_members,omitempty"`
	// Passive healthchecks are disabled by default and can be enabled by attaching a passive health monitor to a server pool. Each time a client connection to a pool member fails, its failed count is incremented. For pools bound to L7 virtual servers, a connection is considered to be failed and failed count is incremented if any TCP connection errors (e.g. TCP RST or failure to send data) or SSL handshake failures occur. For pools bound to L4 virtual servers, if no response is received to a TCP SYN sent to the pool member or if a TCP RST is received in response to a TCP SYN, then the pool member is considered to have failed and the failed count is incremented. The property is deprecated as NSX-T Load Balancer is deprecated. 
	PassiveMonitorPath string `json:"passive_monitor_path,omitempty"`
	SnatTranslation *LbSnatTranslation `json:"snat_translation,omitempty"`
	// TCP multiplexing allows the same TCP connection between load balancer and the backend server to be used for sending multiple client requests from different client TCP connections. The property is deprecated as NSX-T Load Balancer is deprecated. 
	TcpMultiplexingEnabled bool `json:"tcp_multiplexing_enabled,omitempty"`
	// The maximum number of TCP connections per pool that are idly kept alive for sending future client requests. The property is deprecated as NSX-T Load Balancer is deprecated. 
	TcpMultiplexingNumber int64 `json:"tcp_multiplexing_number,omitempty"`
}
