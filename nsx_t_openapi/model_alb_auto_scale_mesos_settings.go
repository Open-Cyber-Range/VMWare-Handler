/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer AutoScaleMesosSettings object
type AlbAutoScaleMesosSettings struct {
	// Apply scaleout even when there are deployments inprogress. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	Force bool `json:"force,omitempty"`
}
