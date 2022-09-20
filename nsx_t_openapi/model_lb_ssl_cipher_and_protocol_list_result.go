/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type LbSslCipherAndProtocolListResult struct {
	// The server will populate this field when returing the resource. Ignored on PUT and POST.
	Links []ResourceLink `json:"_links,omitempty"`
	// Schema for this resource
	Schema string `json:"_schema,omitempty"`
	Self *SelfResourceLink `json:"_self,omitempty"`
	// Opaque cursor to be used for getting next page of records (supplied by current result page)
	Cursor string `json:"cursor,omitempty"`
	// Count of results found (across all pages), set only on first page
	ResultCount int64 `json:"result_count,omitempty"`
	// If true, results are sorted in ascending order
	SortAscending bool `json:"sort_ascending,omitempty"`
	// Field by which records are sorted
	SortBy string `json:"sort_by,omitempty"`
	// List of SSL ciphers
	Ciphers []LbSslCipherInfo `json:"ciphers"`
	// List of SSL protocols
	Protocols []LbSslProtocolInfo `json:"protocols"`
}