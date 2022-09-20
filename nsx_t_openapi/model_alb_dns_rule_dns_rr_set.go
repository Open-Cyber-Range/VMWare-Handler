/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer DnsRuleDnsRrSet object
type AlbDnsRuleDnsRrSet struct {
	ResourceRecordSet *AlbDnsRrSet `json:"resource_record_set"`
	// DNS message section for the resource record set. Enum options - DNS_MESSAGE_SECTION_QUESTION, DNS_MESSAGE_SECTION_ANSWER, DNS_MESSAGE_SECTION_AUTHORITY, DNS_MESSAGE_SECTION_ADDITIONAL. Default value when not specified in API or module is interpreted by ALB Controller as DNS_MESSAGE_SECTION_ANSWER. 
	Section string `json:"section,omitempty"`
}