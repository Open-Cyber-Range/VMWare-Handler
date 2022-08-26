/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer SamlServiceProviderSettings object
type AlbSamlServiceProviderSettings struct {
	// FQDN if entity type is DNS_FQDN .
	Fqdn string `json:"fqdn,omitempty"`
	// Service Provider Organization Display Name.
	OrgDisplayName string `json:"org_display_name,omitempty"`
	// Service Provider Organization Name.
	OrgName string `json:"org_name,omitempty"`
	// Service Provider Organization URL.
	OrgUrl string `json:"org_url,omitempty"`
	// Type of SAML endpoint. Enum options - AUTH_SAML_CLUSTER_VIP, AUTH_SAML_DNS_FQDN, AUTH_SAML_APP_VS. 
	SamlEntityType string `json:"saml_entity_type,omitempty"`
	// Service Provider node information.
	SpNodes []AlbSamlServiceProviderNode `json:"sp_nodes,omitempty"`
	// Service Provider technical contact email.
	TechContactEmail string `json:"tech_contact_email,omitempty"`
	// Service Provider technical contact name.
	TechContactName string `json:"tech_contact_name,omitempty"`
}
