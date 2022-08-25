/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HealthMonitorDNS object
type AlbHealthMonitorDns struct {
	// Query_Type  Response has atleast one answer of which      the resource record type matches the query type   Any_Type  Response should contain atleast one answer  AnyThing  An empty answer is enough. Enum options - DNS_QUERY_TYPE, DNS_ANY_TYPE, DNS_ANY_THING. Default value when not specified in API or module is interpreted by ALB Controller as DNS_QUERY_TYPE. 
	Qtype string `json:"qtype,omitempty"`
	// The DNS monitor will query the DNS server for the fully qualified name in this field. 
	QueryName string `json:"query_name"`
	// When No Error is selected, a DNS query will be marked failed is any error code is returned by the server. With Any selected, the monitor ignores error code in the responses. Enum options - RCODE_NO_ERROR, RCODE_ANYTHING. Default value when not specified in API or module is interpreted by ALB Controller as RCODE_NO_ERROR. 
	Rcode string `json:"rcode,omitempty"`
	// Resource record type used in the healthmonitor DNS query, only A or AAAA type supported. Enum options - DNS_RECORD_OTHER, DNS_RECORD_A, DNS_RECORD_NS, DNS_RECORD_CNAME, DNS_RECORD_SOA, DNS_RECORD_PTR, DNS_RECORD_HINFO, DNS_RECORD_MX, DNS_RECORD_TXT, DNS_RECORD_RP, DNS_RECORD_DNSKEY, DNS_RECORD_AAAA, DNS_RECORD_SRV, DNS_RECORD_OPT, DNS_RECORD_RRSIG, DNS_RECORD_AXFR, DNS_RECORD_ANY. Default value when not specified in API or module is interpreted by ALB Controller as DNS_RECORD_A. 
	RecordType string `json:"record_type,omitempty"`
	// The resource record of the queried DNS server's response for the Request Name must include the IP address defined in this field. 
	ResponseString string `json:"response_string,omitempty"`
}
