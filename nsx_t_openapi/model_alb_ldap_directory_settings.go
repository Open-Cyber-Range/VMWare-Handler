/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer LdapDirectorySettings object
type AlbLdapDirectorySettings struct {
	// LDAP Admin User DN. Administrator credentials are required to search for users under user search DN or groups under group search DN. 
	AdminBindDn string `json:"admin_bind_dn,omitempty"`
	// Group filter is used to identify groups during search. Default value when not specified in API or module is interpreted by ALB Controller as (objectClass=(STAR)). 
	GroupFilter string `json:"group_filter,omitempty"`
	// LDAP group attribute that identifies each of the group members. Default value when not specified in API or module is interpreted by ALB Controller as member. 
	GroupMemberAttribute string `json:"group_member_attribute,omitempty"`
	// Group member entries contain full DNs instead of just user id attribute values. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	GroupMemberIsFullDn bool `json:"group_member_is_full_dn,omitempty"`
	// LDAP group search DN is the root of search for a given group in the LDAP directory. Only matching groups present in this LDAP directory sub-tree will be checked for user membership. 
	GroupSearchDn string `json:"group_search_dn,omitempty"`
	// LDAP group search scope defines how deep to search for the group starting from the group search DN. Enum options - AUTH_LDAP_SCOPE_BASE, AUTH_LDAP_SCOPE_ONE, AUTH_LDAP_SCOPE_SUBTREE. Default value when not specified in API or module is interpreted by ALB Controller as AUTH_LDAP_SCOPE_SUBTREE. 
	GroupSearchScope string `json:"group_search_scope,omitempty"`
	// During user or group search, ignore searching referrals. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	IgnoreReferrals bool `json:"ignore_referrals,omitempty"`
	// LDAP Admin User Password.
	Password string `json:"password,omitempty"`
	// LDAP user attributes to fetch on a successful user bind.
	UserAttributes []string `json:"user_attributes,omitempty"`
	// LDAP user id attribute is the login attribute that uniquely identifies a single user record. 
	UserIdAttribute string `json:"user_id_attribute,omitempty"`
	// LDAP user search DN is the root of search for a given user in the LDAP directory. Only user records present in this LDAP directory sub-tree will be validated. 
	UserSearchDn string `json:"user_search_dn,omitempty"`
	// LDAP user search scope defines how deep to search for the user starting from user search DN. Enum options - AUTH_LDAP_SCOPE_BASE, AUTH_LDAP_SCOPE_ONE, AUTH_LDAP_SCOPE_SUBTREE. Default value when not specified in API or module is interpreted by ALB Controller as AUTH_LDAP_SCOPE_ONE. 
	UserSearchScope string `json:"user_search_scope,omitempty"`
}
