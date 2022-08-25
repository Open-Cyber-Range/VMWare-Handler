/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Base type for CSV result.
type CsvListResult struct {
	// File name set by HTTP server if API  returns CSV result as a file.
	FileName string `json:"file_name,omitempty"`
}
