/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Resource attribute on which constraint should be applied. Example - sourceGroups attribute of Edge CommunicationEntry to be   restricted, is given as:   {      \"target_resource_type\":\"CommunicationEntry\",      \"attribute\":\"sourceGroups\",      \"path_prefix\":\"/infra/domains/vmc-domain/edge-communication-maps/default/communication-entries\"   } 
type ConstraintTarget struct {
	// Attribute name of the target entity.
	Attribute string `json:"attribute,omitempty"`
	// Path prefix of the entity to apply constraint. This is required to further disambiguiate if multiple policy entities share the same resource type. Example - Edge FW and DFW use the same resource type CommunicationMap, CommunicationEntry, Group, etc. 
	PathPrefix string `json:"path_prefix,omitempty"`
	// Resource type of the target entity.
	TargetResourceType string `json:"target_resource_type"`
}
