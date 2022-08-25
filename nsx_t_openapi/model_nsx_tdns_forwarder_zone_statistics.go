/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Statistics counters of the DNS forwarder zone. 
type NsxTdnsForwarderZoneStatistics struct {
	// Domain names configured for the forwarder. Empty if this is the default forwarder. 
	DomainNames []string `json:"domain_names,omitempty"`
	// Statistics per upstream server.
	UpstreamStatistics []NsxTUpstreamServerStatistics `json:"upstream_statistics,omitempty"`
}
