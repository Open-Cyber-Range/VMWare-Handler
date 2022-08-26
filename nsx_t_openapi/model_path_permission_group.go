/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// RBAC Objects qualifier
type PathPermissionGroup struct {
	// Full Object Path
	ObjectPath string `json:"object_path"`
	// Allowed operation
	Operation string `json:"operation"`
}
