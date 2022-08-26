/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Some applications maintain state and require all relevant connections to be sent to the same server as the application state is not synchronized among servers. Persistence is enabled on a LBVirtualServer by binding a persistence profile to it. LBCookiePersistenceProfile is deprecated as NSX-T Load Balancer is deprecated. 
type LbCookiePersistenceProfile struct {
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
	// The resource_type property identifies persistence profile type. LBCookiePersistenceProfile and LBGenericPersistenceProfile are deprecated as NSX-T Load Balancer is deprecated. 
	ResourceType string `json:"resource_type"`
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
	// Persistence shared setting indicates that all LBVirtualServers that consume this LBPersistenceProfile should share the same persistence mechanism when enabled.  Meaning, persistence entries of a client accessing one virtual server will also affect the same client's connections to a different virtual server. For example, say there are two virtual servers vip-ip1:80 and vip-ip1:8080 bound to the same Group g1 consisting of two servers (s11:80 and s12:80). By default, each virtual server will have its own persistence table or cookie. So, in the earlier example, there will be two tables (vip-ip1:80, p1) and (vip-ip1:8080, p1) or cookies. So, if a client connects to vip1:80 and later connects to vip1:8080, the second connection may be sent to a different server than the first.  When persistence_shared is enabled, then the second connection will always connect to the same server as the original connection. For COOKIE persistence type, the same cookie will be shared by multiple virtual servers. For SOURCE_IP persistence type, the persistence table will be shared across virtual servers. For GENERIC persistence type, the persistence table will be shared across virtual servers which consume the same persistence profile in LBRule actions. 
	PersistenceShared bool `json:"persistence_shared,omitempty"`
	// HTTP cookie domain could be configured, only available for insert mode. 
	CookieDomain string `json:"cookie_domain,omitempty"`
	// If fallback is true, once the cookie points to a server that is down (i.e. admin state DISABLED or healthcheck state is DOWN), then a new server is selected by default to handle that request. If fallback is false, it will cause the request to be rejected if cookie points to a server. 
	CookieFallback bool `json:"cookie_fallback,omitempty"`
	// If garble is set to true, cookie value (server IP and port) would be encrypted. If garble is set to false, cookie value would be plain text. 
	CookieGarble bool `json:"cookie_garble,omitempty"`
	// If cookie httponly flag is true, it prevents a script running in the browser from accessing the cookie. Only available for insert mode. 
	CookieHttponly bool `json:"cookie_httponly,omitempty"`
	// Cookie persistence mode.
	CookieMode string `json:"cookie_mode,omitempty"`
	// Cookie name.
	CookieName string `json:"cookie_name,omitempty"`
	// HTTP cookie path could be set, only available for insert mode. 
	CookiePath string `json:"cookie_path,omitempty"`
	// If cookie secure flag is true, it prevents the browser from sending a cookie over http. The cookie is sent only over https. Only available for insert mode. 
	CookieSecure bool `json:"cookie_secure,omitempty"`
	CookieTime *LbCookieTime `json:"cookie_time,omitempty"`
}
