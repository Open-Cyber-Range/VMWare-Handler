/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HSMSafenetClientInfo object
type AlbhsmSafenetClientInfo struct {
	// Generated File - Chrystoki.conf .
	ChrystokiConf string `json:"chrystoki_conf,omitempty"`
	// Client Certificate generated by createCert.
	ClientCert string `json:"client_cert,omitempty"`
	// Name prepended to client key and certificate filename.
	ClientIp string `json:"client_ip"`
	// Client Private Key generated by createCert.
	ClientPrivKey string `json:"client_priv_key,omitempty"`
	// Major number of the sesseion.
	SessionMajorNumber int64 `json:"session_major_number,omitempty"`
	// Minor number of the sesseion.
	SessionMinorNumber int64 `json:"session_minor_number,omitempty"`
}
