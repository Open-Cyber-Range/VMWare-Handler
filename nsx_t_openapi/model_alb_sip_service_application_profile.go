/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer SipServiceApplicationProfile object
type AlbSipServiceApplicationProfile struct {
	// SIP transaction timeout in seconds. Allowed values are 2-512. Unit is SEC. Default value when not specified in API or module is interpreted by ALB Controller as 32. 
	TransactionTimeout int64 `json:"transaction_timeout,omitempty"`
}
