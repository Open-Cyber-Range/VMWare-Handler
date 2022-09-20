/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Identity Firewall user session data list and Directory Group to user mappings. 
type IdfwUserSessionDataAndMappings struct {
	// Active user session data list
	ActiveUserSessions []IdfwUserSessionData `json:"active_user_sessions"`
	// Archived user session data list
	ArchivedUserSessions []IdfwUserSessionData `json:"archived_user_sessions"`
	// Directory Group to user session data mappings
	DirGroupToUserSessionDataMappings []IdfwDirGroupUserSessionMapping `json:"dir_group_to_user_session_data_mappings"`
}