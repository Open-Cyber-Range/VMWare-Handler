/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type TunnelProperties struct {
	// The server will populate this field when returing the resource. Ignored on PUT and POST.
	Links []ResourceLink `json:"_links,omitempty"`
	// Schema for this resource
	Schema string `json:"_schema,omitempty"`
	Self *SelfResourceLink `json:"_self,omitempty"`
	Bfd *BfdProperties `json:"bfd,omitempty"`
	// Corresponds to the interface where local_ip_address is routed.
	EgressInterface string `json:"egress_interface,omitempty"`
	// Tunnel encap
	Encap string `json:"encap,omitempty"`
	// Time at which the Tunnel status has been fetched last time.
	LastUpdatedTime int64 `json:"last_updated_time,omitempty"`
	// Latency type.
	LatencyType string `json:"latency_type,omitempty"`
	// The latency value is set only when latency_type is VALID.
	LatencyValue int64 `json:"latency_value,omitempty"`
	// Local IP address of tunnel
	LocalIp string `json:"local_ip,omitempty"`
	// Name of tunnel
	Name string `json:"name,omitempty"`
	// Remote IP address of tunnel
	RemoteIp string `json:"remote_ip,omitempty"`
	// Represents the display name of the remote transport node at the other end of the tunnel.
	RemoteNodeDisplayName string `json:"remote_node_display_name,omitempty"`
	// UUID of the remote transport node
	RemoteNodeId string `json:"remote_node_id,omitempty"`
	// Status of tunnel
	Status string `json:"status,omitempty"`
}