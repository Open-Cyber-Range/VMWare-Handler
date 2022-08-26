/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Parameter to force switch over from Standby to Active. 
type GlobalManagerSwitchOverRequestParameter struct {
	// If true indicates that user requested make standby Global Manager as active ignoring the state of current active Global Manager. Typically, recommended to use when active Global Manager is failed or not reachable. 
	Force bool `json:"force,omitempty"`
}
