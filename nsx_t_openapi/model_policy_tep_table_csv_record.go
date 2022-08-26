/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type PolicyTepTableCsvRecord struct {
	// This is the identifier of the TEP segment. This segment is NOT the same as logical segment or logical switch.
	SegmentId string `json:"segment_id,omitempty"`
	// The tunnel endpoint IP address
	TepIp string `json:"tep_ip,omitempty"`
	// The tunnel endpoint label
	TepLabel int64 `json:"tep_label"`
	// The tunnel endpoint MAC address
	TepMacAddress string `json:"tep_mac_address"`
}
