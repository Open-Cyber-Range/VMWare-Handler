/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// L2VPN transport tunnel peer code.
type L2VpnSessionTransportTunnelPeerCode struct {
	// Peer code represents a base64 encoded string which has all the configuration for tunnel. E.g local/peer ips and protocol, encryption algorithm, etc. Peer code also contains PSK; be careful when sharing or storing it. 
	PeerCode string `json:"peer_code,omitempty"`
	// Policy Path referencing the transport tunnel.
	TransportTunnelPath string `json:"transport_tunnel_path,omitempty"`
}
