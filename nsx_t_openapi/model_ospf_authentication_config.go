/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Enables OSPF authentication with specified mode and password.
type OspfAuthenticationConfig struct {
	// Authentication secret key id is mandatory for type md5 with min value of 1 and max value 255. 
	KeyId int64 `json:"key_id,omitempty"`
	// If mode is MD5 or PASSWORD, Authentication secret key is mandatory if mode is NONE, then authentication is disabled. 
	Mode string `json:"mode,omitempty"`
	// Authentication secret is mandatory for type password and md5 with min length of 1 and max length 8. 
	SecretKey string `json:"secret_key,omitempty"`
}