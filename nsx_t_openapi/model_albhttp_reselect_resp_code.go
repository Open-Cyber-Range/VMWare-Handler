/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer HTTPReselectRespCode object
type AlbhttpReselectRespCode struct {
	// HTTP response code to be matched. Allowed values are 400-599. 
	Codes []int64 `json:"codes,omitempty"`
	// HTTP response code ranges to match.
	Ranges []AlbhttpStatusRange `json:"ranges,omitempty"`
	// Block of HTTP response codes to match for server reselect. Enum options - HTTP_RSP_4XX, HTTP_RSP_5XX. 
	RespCodeBlock []string `json:"resp_code_block,omitempty"`
}
