/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// List of Users logged into VMs where intrusions of a given signature were detected. 
type PolicyIdsUserStats struct {
	// Number of unique users logged into VMs on which a particular signature was detected.
	Count int64 `json:"count,omitempty"`
	// List of users logged into VMs on which a particular signature was detected.
	UserList []string `json:"user_list,omitempty"`
}