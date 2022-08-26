/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// API Request Body is an Event Source that represents an API request body that is being reveived as part of an API. Supported Request Bodies are those received as part of a PATCH/PUT/POST request. 
type ApiRequestBody struct {
	// Event Source resource type. 
	ResourceType string `json:"resource_type"`
	// Regex path representing a regex expression on resources. This regex is used to identify the request body(ies) that is/are the source of the Event. For instance: specifying \"Lb* | /infra/tier-0s/vmc/ipsec-vpn-services/default\" as a source means that ANY resource starting with Lb or ANY resource with \"/infra/tier-0s/vmc/ipsec-vpn-services/default\" as path would be the source of the event in question. 
	ResourcePointer string `json:"resource_pointer"`
}
