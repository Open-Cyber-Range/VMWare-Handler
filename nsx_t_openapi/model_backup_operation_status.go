/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Backup operation status
type BackupOperationStatus struct {
	// Unique identifier of a backup
	BackupId string `json:"backup_id"`
	// Time when operation was ended
	EndTime int64 `json:"end_time,omitempty"`
	// Error code
	ErrorCode string `json:"error_code,omitempty"`
	// Error code details
	ErrorMessage string `json:"error_message,omitempty"`
	// Time when operation was started
	StartTime int64 `json:"start_time,omitempty"`
	// True if backup is successfully completed, else false
	Success bool `json:"success"`
}
