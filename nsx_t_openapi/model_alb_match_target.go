/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer MatchTarget object
type AlbMatchTarget struct {
	ClientIp *AlbIpAddrMatch `json:"client_ip,omitempty"`
	Cookie *AlbCookieMatch `json:"cookie,omitempty"`
	// Configure HTTP header(s).
	Hdrs []AlbHdrMatch `json:"hdrs,omitempty"`
	HostHdr *AlbHostHdrMatch `json:"host_hdr,omitempty"`
	Method *AlbMethodMatch `json:"method,omitempty"`
	Path *AlbPathMatch `json:"path,omitempty"`
	Protocol *AlbProtocolMatch `json:"protocol,omitempty"`
	Query *AlbQueryMatch `json:"query,omitempty"`
	Version *AlbhttpVersionMatch `json:"version,omitempty"`
	VsPort *AlbPortMatch `json:"vs_port,omitempty"`
}
