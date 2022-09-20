/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type Ipv6Header struct {
	// The destination ip address.
	DstIp string `json:"dst_ip,omitempty"`
	// Decremented by 1 by each node that forwards the packets. The packet is discarded if Hop Limit is decremented to zero.
	HopLimit int64 `json:"hop_limit,omitempty"`
	// Identifies the type of header immediately following the IPv6 header.
	NextHeader int64 `json:"next_header,omitempty"`
	// The source ip address.
	SrcIp string `json:"src_ip,omitempty"`
}