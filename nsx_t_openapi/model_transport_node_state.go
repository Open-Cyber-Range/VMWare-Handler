/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Transport Node State
type TransportNodeState struct {
	// Array of configuration state of various sub systems
	Details []ConfigurationStateElement `json:"details,omitempty"`
	// Error code
	FailureCode int64 `json:"failure_code,omitempty"`
	// Error message in case of failure
	FailureMessage string `json:"failure_message,omitempty"`
	// Gives details of state of desired configuration. Additional enums with more details on progress/success/error states are sent for edge node. The success states are NODE_READY and TRANSPORT_NODE_READY, pending states are {VM_DEPLOYMENT_QUEUED, VM_DEPLOYMENT_IN_PROGRESS, REGISTRATION_PENDING} and other values indicate failures. \"in_sync\" state indicates that the desired configuration has been received by the host to which it applies, but is not yet in effect. When the configuration is actually in effect, the state will change to \"success\". Please note, failed state is deprecated. 
	State string `json:"state,omitempty"`
	DeploymentProgressState *TransportNodeDeploymentProgressState `json:"deployment_progress_state,omitempty"`
	// States of HostSwitches on the host
	HostSwitchStates []HostSwitchState `json:"host_switch_states,omitempty"`
	// the present realized maintenance mode state
	MaintenanceModeState string `json:"maintenance_mode_state,omitempty"`
	NodeDeploymentState *ConfigurationState `json:"node_deployment_state,omitempty"`
	RemoteTunnelEndpointState *RemoteTunnelEndpointConfigState `json:"remote_tunnel_endpoint_state,omitempty"`
	// Unique Id of the TransportNode
	TransportNodeId string `json:"transport_node_id,omitempty"`
}
