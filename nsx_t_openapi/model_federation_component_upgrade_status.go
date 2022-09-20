/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

type FederationComponentUpgradeStatus struct {
	// Component type for the upgrade status
	ComponentType string `json:"component_type,omitempty"`
	// Mapping of current versions of nodes and counts of nodes at the respective versions.
	CurrentVersionNodeSummary []FederationNodeSummary `json:"current_version_node_summary,omitempty"`
	// Details about the upgrade status
	Details string `json:"details,omitempty"`
	// Indicator of upgrade progress in percentage
	PercentComplete float32 `json:"percent_complete,omitempty"`
	// Upgrade status of component
	Status string `json:"status,omitempty"`
	// Target component version
	TargetVersion string `json:"target_version,omitempty"`
}