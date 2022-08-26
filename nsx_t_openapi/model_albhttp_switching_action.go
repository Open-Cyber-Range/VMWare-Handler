/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HTTPSwitchingAction object
type AlbhttpSwitchingAction struct {
	// Content switching action type. Enum options - HTTP_SWITCHING_SELECT_POOL, HTTP_SWITCHING_SELECT_LOCAL, HTTP_SWITCHING_SELECT_POOLGROUP. Allowed in Essentials(Allowed values- HTTP_SWITCHING_SELECT_POOL,HTTP_SWITCHING_SELECT_LOCAL) edition, Enterprise edition. 
	Action string `json:"action"`
	File *AlbhttpLocalFile `json:"file,omitempty"`
	// path of the pool group to serve the request. It is a reference to an object of type PoolGroup. 
	PoolGroupPath string `json:"pool_group_path,omitempty"`
	// path of the pool of servers to serve the request. It is a reference to an object of type Pool. 
	PoolPath string `json:"pool_path,omitempty"`
	Server *AlbPoolServer `json:"server,omitempty"`
	// HTTP status code to use when serving local response. Enum options - HTTP_LOCAL_RESPONSE_STATUS_CODE_200, HTTP_LOCAL_RESPONSE_STATUS_CODE_204, HTTP_LOCAL_RESPONSE_STATUS_CODE_403, HTTP_LOCAL_RESPONSE_STATUS_CODE_404, HTTP_LOCAL_RESPONSE_STATUS_CODE_429, HTTP_LOCAL_RESPONSE_STATUS_CODE_501. 
	StatusCode string `json:"status_code,omitempty"`
}
