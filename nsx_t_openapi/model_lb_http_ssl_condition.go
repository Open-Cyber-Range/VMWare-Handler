/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// This condition is used to match SSL handshake and SSL connection at all phases.If multiple properties are configured, the rule is considered a match when all the configured properties are matched. 
type LbHttpSslCondition struct {
	// A flag to indicate whether reverse the match result of this condition
	Inverse bool `json:"inverse,omitempty"`
	// Type of load balancer rule condition
	Type_ string `json:"type"`
	ClientCertificateIssuerDn *LbClientCertificateIssuerDnCondition `json:"client_certificate_issuer_dn,omitempty"`
	ClientCertificateSubjectDn *LbClientCertificateSubjectDnCondition `json:"client_certificate_subject_dn,omitempty"`
	// Cipher list which supported by client.
	ClientSupportedSslCiphers []string `json:"client_supported_ssl_ciphers,omitempty"`
	// The type of SSL session reused.
	SessionReused string `json:"session_reused,omitempty"`
	// Protocol of an established SSL connection.
	UsedProtocol string `json:"used_protocol,omitempty"`
	// Cipher used for an established SSL connection.
	UsedSslCipher string `json:"used_ssl_cipher,omitempty"`
}