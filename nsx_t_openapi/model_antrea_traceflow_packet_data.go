/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Packet data stuffs for Antrea traceflow.
type AntreaTraceflowPacketData struct {
	// This property is used to set packet data size. 
	FrameSize int64 `json:"frameSize,omitempty"`
	IpHeader *AntreaTraceflowIpHeader `json:"ipHeader,omitempty"`
	Ipv6Header *AntreaTraceflowIpv6Header `json:"ipv6Header,omitempty"`
	// This property is used to set payload data. 
	Payload string `json:"payload,omitempty"`
	// This property is used to set resource type. 
	ResourceType string `json:"resourceType,omitempty"`
	TransportHeader *AntreaTraceflowTransportHeader `json:"transportHeader,omitempty"`
	// This property is used to set transport type. 
	TransportType string `json:"transportType,omitempty"`
}
