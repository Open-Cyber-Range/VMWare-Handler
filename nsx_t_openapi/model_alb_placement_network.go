/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer PlacementNetwork object
type AlbPlacementNetwork struct {
	// It is a reference to an object of type Network.
	NetworkName string `json:"network_name"`
	Subnet *AlbIpAddrPrefix `json:"subnet"`
}