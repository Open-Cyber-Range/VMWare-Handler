/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Traffic statistics for a segment.
type L2VpnTrafficStatisticsPerSegment struct {
	// Total number of incoming Broadcast, Unknown unicast and Multicast (BUM) bytes. 
	BumBytesIn int64 `json:"bum_bytes_in,omitempty"`
	// Total number of outgoing Broadcast, Unknown unicast and Multicast (BUM) bytes. 
	BumBytesOut int64 `json:"bum_bytes_out,omitempty"`
	// Total number of incoming Broadcast, Unknown unicast and Multicast (BUM) packets. 
	BumPacketsIn int64 `json:"bum_packets_in,omitempty"`
	// Total number of outgoing Broadcast, Unknown unicast and Multicast (BUM) packets. 
	BumPacketsOut int64 `json:"bum_packets_out,omitempty"`
	// Total number of incoming bytes. 
	BytesIn int64 `json:"bytes_in,omitempty"`
	// Total number of outgoing bytes. 
	BytesOut int64 `json:"bytes_out,omitempty"`
	// Total number of incoming packets. 
	PacketsIn int64 `json:"packets_in,omitempty"`
	// Total number of outgoing packets. 
	PacketsOut int64 `json:"packets_out,omitempty"`
	// Total number of incoming packets dropped. 
	PacketsReceiveError int64 `json:"packets_receive_error,omitempty"`
	// Total number of packets dropped while sending for any reason. 
	PacketsSentError int64 `json:"packets_sent_error,omitempty"`
	// Policy path referencing the segment on which stats are gathered. 
	SegmentPath string `json:"segment_path,omitempty"`
}
