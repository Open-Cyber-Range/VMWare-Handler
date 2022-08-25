/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Segment statistics on specific Enforcement Point.
type SegmentStatistics struct {
	RxBytes *DataCounter `json:"rx_bytes,omitempty"`
	RxPackets *DataCounter `json:"rx_packets,omitempty"`
	TxBytes *DataCounter `json:"tx_bytes,omitempty"`
	TxPackets *DataCounter `json:"tx_packets,omitempty"`
	DroppedBySecurityPackets *PacketsDroppedBySecurity `json:"dropped_by_security_packets,omitempty"`
	MacLearning *MacLearningCounters `json:"mac_learning,omitempty"`
	// Timestamp when the data was last updated; unset if data source has never updated the data.
	LastUpdateTimestamp int64 `json:"last_update_timestamp,omitempty"`
	// The id of the logical Switch
	LogicalSwitchId string `json:"logical_switch_id,omitempty"`
}
