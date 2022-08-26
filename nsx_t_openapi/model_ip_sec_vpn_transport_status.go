/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Provides IPSec VPN session status.
type IpSecVpnTransportStatus struct {
	ResourceType string `json:"resource_type"`
	// Policy path referencing Transport Tunnel.
	TransportTunnelPath string `json:"transport_tunnel_path,omitempty"`
	SessionStatus *IpSecVpnSessionStatusNsxT `json:"session_status,omitempty"`
}
