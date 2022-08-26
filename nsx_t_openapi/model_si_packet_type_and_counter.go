/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type SiPacketTypeAndCounter struct {
	// The number of packets.
	Counter int64 `json:"counter"`
	// The type of the packets
	PacketType string `json:"packet_type"`
}
