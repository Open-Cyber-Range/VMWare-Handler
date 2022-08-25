/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Arbitrary key-value pairs that may be attached to an entity
type Tag struct {
	// Tag searches may optionally be restricted by scope
	Scope string `json:"scope,omitempty"`
	// Identifier meaningful to user with maximum length of 256 characters
	Tag string `json:"tag,omitempty"`
}
