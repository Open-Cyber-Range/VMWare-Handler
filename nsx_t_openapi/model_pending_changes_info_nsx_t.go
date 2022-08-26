/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Information about recent changes, if any, that are not reflected in the Enforced Realized Status. 
type PendingChangesInfoNsxT struct {
	// Flag describing whether there are any pending changes that are not reflected in the status. 
	PendingChangesFlag bool `json:"pending_changes_flag,omitempty"`
}
