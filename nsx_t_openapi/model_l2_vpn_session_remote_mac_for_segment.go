/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Remote MAC addresses for logical switch.
type L2VpnSessionRemoteMacForSegment struct {
	// Remote Mac addresses.
	RemoteMacAddresses []string `json:"remote_mac_addresses,omitempty"`
	// Intent path of the segment.
	SegmentPath string `json:"segment_path"`
}