/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type MacTableEntry struct {
	// The MAC address
	MacAddress string `json:"mac_address"`
	// RTEP group id is applicable when the logical switch is stretched across multiple sites. When rtep_group_id is set, mac_address represents remote mac_address. 
	RtepGroupId int64 `json:"rtep_group_id,omitempty"`
	// VTEP group id is applicable when the logical switch is stretched across multiple sites. When vtep_group_id is set, mac_address represents remote mac_address. 
	VtepGroupId int64 `json:"vtep_group_id,omitempty"`
	// The virtual tunnel endpoint IP address
	VtepIp string `json:"vtep_ip,omitempty"`
	// The virtual tunnel endpoint MAC address
	VtepMacAddress string `json:"vtep_mac_address,omitempty"`
}
