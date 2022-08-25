/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type ArpHeader struct {
	// The destination IP address
	DstIp string `json:"dst_ip"`
	// This field specifies the nature of the Arp message being sent.
	OpCode string `json:"op_code"`
	// This field specifies the IP address of the sender. If omitted, the src_ip is set to 0.0.0.0.
	SrcIp string `json:"src_ip,omitempty"`
}
