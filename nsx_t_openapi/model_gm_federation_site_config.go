/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Additional configuration required for federation at Site.
type GmFederationSiteConfig struct {
	// IP Addresses to be allocated for transit segment when the gateway is stretched. Note that Global Manager will carve out the IP Pool for each site to be used for edge nodes when gateway is stretched based on the user provided subnet and maximum number of edge nodes allowed per site. 
	TransitSubnet string `json:"transit_subnet,omitempty"`
}
