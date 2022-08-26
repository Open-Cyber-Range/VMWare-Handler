/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HealthMonitorSIP object
type AlbHealthMonitorSip struct {
	// Specify the transport protocol TCP or UDP, to be used for SIP health monitor. The default transport is UDP. Enum options - SIP_UDP_PROTO, SIP_TCP_PROTO. Default value when not specified in API or module is interpreted by ALB Controller as SIP_UDP_PROTO. 
	SipMonitorTransport string `json:"sip_monitor_transport,omitempty"`
	// Specify the SIP request to be sent to the server. By default, SIP OPTIONS request will be sent. Enum options - SIP_OPTIONS. Default value when not specified in API or module is interpreted by ALB Controller as SIP_OPTIONS. 
	SipRequestCode string `json:"sip_request_code,omitempty"`
	// Match for a keyword in the first 2KB of the server header and body response. By default, it matches for SIP/2.0. Default value when not specified in API or module is interpreted by ALB Controller as SIP/2.0. 
	SipResponse string `json:"sip_response,omitempty"`
}
