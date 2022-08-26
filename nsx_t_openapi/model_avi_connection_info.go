/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Credential info to connect to a AVI type of enforcement point.
type AviConnectionInfo struct {
	// Value of this property could be Hostname or IP. For instance: - On an NSX-T MP running on default port, the value could be \"10.192.1.1\" - On an NSX-T MP running on custom port, the value could be \"192.168.1.1:32789\" - On an NSX-T MP in VMC deployments, the value could be \"192.168.1.1:5480/nsxapi\" 
	EnforcementPointAddress string `json:"enforcement_point_address"`
	// Resource Type of Enforcement Point Connection Info.
	ResourceType string `json:"resource_type"`
	// Clouds are containers for the environment that Avi Vantage is installed or operating within. During initial setup of Vantage, a default cloud, named Default-Cloud, is created. This is where the first Controller is deployed, into Default-Cloud. Additional clouds may be added, containing SEs and virtual services. This is a deprecated property. Cloud has been renamed to cloud_name and it will added from specific ALB entity. 
	Cloud string `json:"cloud,omitempty"`
	// Expiry time of the token will be set by LCM at the time of Enforcement Point Creation. 
	ExpiresAt string `json:"expires_at,omitempty"`
	// Managed by used when on-borading workflow created by LCM/VCF. 
	ManagedBy string `json:"managed_by,omitempty"`
	// Password or Token for Avi Controller.
	Password string `json:"password,omitempty"`
	// A tenant is an isolated instance of Avi Controller. Each Avi user account is associated with one or more tenants. The tenant associated with a user account defines the resources that user can access within Avi Vantage. When a user logs in, Avi restricts their access to only those resources that are in the same tenant 
	Tenant string `json:"tenant"`
	// Thumbprint of EnforcementPoint in the form of a SHA-256 hash represented in lower case HEX. 
	Thumbprint string `json:"thumbprint,omitempty"`
	// Username.
	Username string `json:"username,omitempty"`
	// Avi supports API versioning for backward compatibility with automation scripts written for an object model older than the current one. Such scripts need not be updated to keep up with object model changes This is a deprecated property. The version is now auto populated from property file and its value can be read using APIs 
	Version string `json:"version,omitempty"`
}
