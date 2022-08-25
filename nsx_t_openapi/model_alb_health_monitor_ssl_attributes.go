/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HealthMonitorSSLAttributes object
type AlbHealthMonitorSslAttributes struct {
	// PKI profile used to validate the SSL certificate presented by a server. It is a reference to an object of type PKIProfile. 
	PkiProfilePath string `json:"pki_profile_path,omitempty"`
	// Fully qualified DNS hostname which will be used in the TLS SNI extension in server connections indicating SNI is enabled. 
	ServerName string `json:"server_name,omitempty"`
	// Service engines will present this SSL certificate to the server. It is a reference to an object of type SSLKeyAndCertificate. 
	SslKeyAndCertificatePath string `json:"ssl_key_and_certificate_path,omitempty"`
	// SSL profile defines ciphers and SSL versions to be used for healthmonitor traffic to the back-end servers. It is a reference to an object of type SSLProfile. 
	SslProfilePath string `json:"ssl_profile_path"`
}
