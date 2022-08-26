/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Event log server connection status
type DirectoryEventLogServerStatus struct {
	// Additional optional detail error message
	ErrorMessage string `json:"error_message,omitempty"`
	// Last event record ID is an opaque integer value that shows the last successfully received event from event log server.
	LastEventRecordId int64 `json:"last_event_record_id,omitempty"`
	// Time of last successfully received and record event from event log server.
	LastEventTimeCreated int64 `json:"last_event_time_created,omitempty"`
	// Last polling time
	LastPollingTime int64 `json:"last_polling_time,omitempty"`
	// Connection status:     OK: All OK     ERROR: Generic error 
	Status string `json:"status,omitempty"`
}
