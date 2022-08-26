/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// The traffic_name specifies the infrastructure traffic type and it must be one of the following system-defined types: FAULT_TOLERANCE is traffic for failover and recovery. HBR is traffic for Host based replication. ISCSI is traffic for Internet Small Computer System Interface. MANAGEMENT is traffic for host management. NFS is traffic related to file transfer in network file system. VDP is traffic for vSphere data protection. VIRTUAL_MACHINE is traffic generated by virtual machines. VMOTION is traffic for computing resource migration. VSAN is traffic generated by virtual storage area network. The dynamic_res_pool_name provides a name for the resource pool. It can be any arbitrary string. Either traffic_name or dynamic_res_pool_name must be set. If both are specified or omitted, an error will be returned. 
type PolicyHostInfraTrafficType struct {
	// Dynamic resource pool traffic name
	DynamicResPoolName string `json:"dynamic_res_pool_name,omitempty"`
	// Traffic types
	TrafficName string `json:"traffic_name,omitempty"`
}
