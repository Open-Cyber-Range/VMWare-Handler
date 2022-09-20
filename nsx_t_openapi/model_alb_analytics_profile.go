/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Advanced load balancer AnalyticsProfile object
type AlbAnalyticsProfile struct {
	// The server will populate this field when returing the resource. Ignored on PUT and POST.
	Links []ResourceLink `json:"_links,omitempty"`
	// Schema for this resource
	Schema string `json:"_schema,omitempty"`
	Self *SelfResourceLink `json:"_self,omitempty"`
	// The _revision property describes the current revision of the resource. To prevent clients from overwriting each other's changes, PUT operations must include the current _revision of the resource, which clients should obtain by issuing a GET operation. If the _revision provided in a PUT request is missing or stale, the operation will be rejected.
	Revision int32 `json:"_revision,omitempty"`
	// Timestamp of resource creation
	CreateTime int64 `json:"_create_time,omitempty"`
	// ID of the user who created this resource
	CreateUser string `json:"_create_user,omitempty"`
	// Timestamp of last modification
	LastModifiedTime int64 `json:"_last_modified_time,omitempty"`
	// ID of the user who last modified this resource
	LastModifiedUser string `json:"_last_modified_user,omitempty"`
	// Protection status is one of the following: PROTECTED - the client who retrieved the entity is not allowed             to modify it. NOT_PROTECTED - the client who retrieved the entity is allowed                 to modify it REQUIRE_OVERRIDE - the client who retrieved the entity is a super                    user and can modify it, but only when providing                    the request header X-Allow-Overwrite=true. UNKNOWN - the _protection field could not be determined for this           entity. 
	Protection string `json:"_protection,omitempty"`
	// Indicates system owned resource
	SystemOwned bool `json:"_system_owned,omitempty"`
	// Description of this resource
	Description string `json:"description,omitempty"`
	// Defaults to ID if not set
	DisplayName string `json:"display_name,omitempty"`
	// Unique identifier of this resource
	Id string `json:"id,omitempty"`
	// The type of this resource.
	ResourceType string `json:"resource_type,omitempty"`
	// Opaque identifiers meaningful to the API user
	Tags []Tag `json:"tags,omitempty"`
	// Path of its parent
	ParentPath string `json:"parent_path,omitempty"`
	// Absolute path of this object
	Path string `json:"path,omitempty"`
	// This is a UUID generated by the system for realizing the entity object. In most cases this should be same as 'unique_id' of the entity. However, in some cases this can be different because of entities have migrated thier unique identifier to NSX Policy intent objects later in the timeline and did not use unique_id for realization. Realization id is helpful for users to debug data path to correlate the configuration with corresponding intent. 
	RealizationId string `json:"realization_id,omitempty"`
	// Path relative from its parent
	RelativePath string `json:"relative_path,omitempty"`
	// This is a UUID generated by the GM/LM to uniquely identify entites in a federated environment. For entities that are stretched across multiple sites, the same ID will be used on all the stretched sites. 
	UniqueId string `json:"unique_id,omitempty"`
	// subtree for this type within policy tree containing nested elements. 
	Children []ChildPolicyConfigResource `json:"children,omitempty"`
	// Intent objects are not directly deleted from the system when a delete is invoked on them. They are marked for deletion and only when all the realized entities for that intent object gets deleted, the intent object is deleted. Objects that are marked for deletion are not returned in GET call. One can use the search API to get these objects. 
	MarkedForDelete bool `json:"marked_for_delete,omitempty"`
	// Global intent objects cannot be modified by the user. However, certain global intent objects can be overridden locally by use of this property. In such cases, the overridden local values take precedence over the globally defined values for the properties. 
	Overridden bool `json:"overridden,omitempty"`
	// If a client receives an HTTP response in less than the Satisfactory Latency Threshold, the request is considered Satisfied. It is considered Tolerated if it is not Satisfied and less than Tolerated Latency Factor multiplied by the Satisfactory Latency Threshold. Greater than this number and the client's request is considered Frustrated. Allowed values are 1-30000. Unit is MILLISECONDS. Allowed in Basic(Allowed values- 500) edition, Essentials(Allowed values- 500) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 500. 
	ApdexResponseThreshold int64 `json:"apdex_response_threshold,omitempty"`
	// Client tolerated response latency factor. Client must receive a response within this factor times the satisfactory threshold (apdex_response_threshold) to be considered tolerated. Allowed values are 1-1000. Allowed in Basic(Allowed values- 4) edition, Essentials(Allowed values- 4) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 4.0. 
	ApdexResponseToleratedFactor float32 `json:"apdex_response_tolerated_factor,omitempty"`
	// Satisfactory client to Avi Round Trip Time(RTT). Allowed values are 1-2000. Unit is MILLISECONDS. Allowed in Basic(Allowed values- 250) edition, Essentials(Allowed values- 250) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 250. 
	ApdexRttThreshold int64 `json:"apdex_rtt_threshold,omitempty"`
	// Tolerated client to Avi Round Trip Time(RTT) factor. It is a multiple of apdex_rtt_tolerated_factor. Allowed values are 1-1000. Allowed in Basic(Allowed values- 4) edition, Essentials(Allowed values- 4) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 4.0. 
	ApdexRttToleratedFactor float32 `json:"apdex_rtt_tolerated_factor,omitempty"`
	// If a client is able to load a page in less than the Satisfactory Latency Threshold, the PageLoad is considered Satisfied. It is considered tolerated if it is greater than Satisfied but less than the Tolerated Latency multiplied by Satisifed Latency. Greater than this number and the client's request is considered Frustrated. A PageLoad includes the time for DNS lookup, download of all HTTP objects, and page render time. Allowed values are 1-30000. Unit is MILLISECONDS. Allowed in Basic(Allowed values- 5000) edition, Essentials(Allowed values- 5000) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 5000. 
	ApdexRumThreshold int64 `json:"apdex_rum_threshold,omitempty"`
	// Virtual service threshold factor for tolerated Page Load Time (PLT) as multiple of apdex_rum_threshold. Allowed values are 1-1000. Allowed in Basic(Allowed values- 4) edition, Essentials(Allowed values- 4) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 4.0. 
	ApdexRumToleratedFactor float32 `json:"apdex_rum_tolerated_factor,omitempty"`
	// A server HTTP response is considered Satisfied if latency is less than the Satisfactory Latency Threshold. The response is considered tolerated when it is greater than Satisfied but less than the Tolerated Latency Factor (STAR) S_Latency. Greater than this number and the server response is considered Frustrated. Allowed values are 1-30000. Unit is MILLISECONDS. Allowed in Basic(Allowed values- 400) edition, Essentials(Allowed values- 400) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 400. 
	ApdexServerResponseThreshold int64 `json:"apdex_server_response_threshold,omitempty"`
	// Server tolerated response latency factor. Servermust response within this factor times the satisfactory threshold (apdex_server_response_threshold) to be considered tolerated. Allowed values are 1-1000. Allowed in Basic(Allowed values- 4) edition, Essentials(Allowed values- 4) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 4.0. 
	ApdexServerResponseToleratedFactor float32 `json:"apdex_server_response_tolerated_factor,omitempty"`
	// Satisfactory client to Avi Round Trip Time(RTT). Allowed values are 1-2000. Unit is MILLISECONDS. Allowed in Basic(Allowed values- 125) edition, Essentials(Allowed values- 125) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 125. 
	ApdexServerRttThreshold int64 `json:"apdex_server_rtt_threshold,omitempty"`
	// Tolerated client to Avi Round Trip Time(RTT) factor. It is a multiple of apdex_rtt_tolerated_factor. Allowed values are 1-1000. Allowed in Basic(Allowed values- 4) edition, Essentials(Allowed values- 4) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 4.0. 
	ApdexServerRttToleratedFactor float32 `json:"apdex_server_rtt_tolerated_factor,omitempty"`
	ClientLogConfig *AlbClientLogConfiguration `json:"client_log_config,omitempty"`
	ClientLogStreamingConfig *AlbClientLogStreamingConfig `json:"client_log_streaming_config,omitempty"`
	// A connection between client and Avi is considered lossy when more than this percentage of out of order packets are received. Allowed values are 1-100. Unit is PERCENT. Allowed in Basic(Allowed values- 50) edition, Essentials(Allowed values- 50) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 50. 
	ConnLossyOooThreshold int64 `json:"conn_lossy_ooo_threshold,omitempty"`
	// A connection between client and Avi is considered lossy when more than this percentage of packets are retransmitted due to timeout. Allowed values are 1-100. Unit is PERCENT. Allowed in Basic(Allowed values- 20) edition, Essentials(Allowed values- 20) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 20. 
	ConnLossyTimeoRexmtThreshold int64 `json:"conn_lossy_timeo_rexmt_threshold,omitempty"`
	// A connection between client and Avi is considered lossy when more than this percentage of packets are retransmitted. Allowed values are 1-100. Unit is PERCENT. Allowed in Basic(Allowed values- 50) edition, Essentials(Allowed values- 50) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 50. 
	ConnLossyTotalRexmtThreshold int64 `json:"conn_lossy_total_rexmt_threshold,omitempty"`
	// A client connection is considered lossy when percentage of times a packet could not be trasmitted due to TCP zero window is above this threshold. Allowed values are 0-100. Unit is PERCENT. Allowed in Basic(Allowed values- 2) edition, Essentials(Allowed values- 2) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 2. 
	ConnLossyZeroWinSizeEventThreshold int64 `json:"conn_lossy_zero_win_size_event_threshold,omitempty"`
	// A connection between Avi and server is considered lossy when more than this percentage of out of order packets are received. Allowed values are 1-100. Unit is PERCENT. Allowed in Basic(Allowed values- 50) edition, Essentials(Allowed values- 50) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 50. 
	ConnServerLossyOooThreshold int64 `json:"conn_server_lossy_ooo_threshold,omitempty"`
	// A connection between Avi and server is considered lossy when more than this percentage of packets are retransmitted due to timeout. Allowed values are 1-100. Unit is PERCENT. Allowed in Basic(Allowed values- 20) edition, Essentials(Allowed values- 20) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 20. 
	ConnServerLossyTimeoRexmtThreshold int64 `json:"conn_server_lossy_timeo_rexmt_threshold,omitempty"`
	// A connection between Avi and server is considered lossy when more than this percentage of packets are retransmitted. Allowed values are 1-100. Unit is PERCENT. Allowed in Basic(Allowed values- 50) edition, Essentials(Allowed values- 50) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 50. 
	ConnServerLossyTotalRexmtThreshold int64 `json:"conn_server_lossy_total_rexmt_threshold,omitempty"`
	// A server connection is considered lossy when percentage of times a packet could not be trasmitted due to TCP zero window is above this threshold. Allowed values are 0-100. Unit is PERCENT. Allowed in Basic(Allowed values- 2) edition, Essentials(Allowed values- 2) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 2. 
	ConnServerLossyZeroWinSizeEventThreshold int64 `json:"conn_server_lossy_zero_win_size_event_threshold,omitempty"`
	// Enable adaptive configuration for optimizing resource usage. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	EnableAdaptiveConfig bool `json:"enable_adaptive_config,omitempty"`
	// Enables Advanced Analytics features like Anomaly detection. If set to false, anomaly computation (and associated rules/events) for VS, Pool and Server metrics will be deactivated. However, setting it to false reduces cpu and memory requirements for Analytics subsystem. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Special default for Basic edition is false, Essentials edition is false, Enterprise is True. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	EnableAdvancedAnalytics bool `json:"enable_advanced_analytics,omitempty"`
	// Virtual Service (VS) metrics are processed only when there is live data traffic on the VS. In case, VS is idle for a period of time as specified by ondemand_metrics_idle_timeout then metrics processing is suspended for that VS. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	EnableOndemandMetrics bool `json:"enable_ondemand_metrics,omitempty"`
	// Enable node (service engine) level analytics forvs metrics. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	EnableSeAnalytics bool `json:"enable_se_analytics,omitempty"`
	// Enables analytics on backend servers. This may be desired in container environment when there are large number of ephemeral servers. Additionally, no healthscore of servers is computed when server analytics is enabled. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	EnableServerAnalytics bool `json:"enable_server_analytics,omitempty"`
	// Enable VirtualService (frontend) Analytics. This flag enables metrics and healthscore for Virtualservice. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	EnableVsAnalytics bool `json:"enable_vs_analytics,omitempty"`
	// Exclude client closed connection before an HTTP request could be completed from being classified as an error. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeClientCloseBeforeRequestAsError bool `json:"exclude_client_close_before_request_as_error,omitempty"`
	// Exclude dns policy drops from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeDnsPolicyDropAsSignificant bool `json:"exclude_dns_policy_drop_as_significant,omitempty"`
	// Exclude queries to GSLB services that are operationally down from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeGsDownAsError bool `json:"exclude_gs_down_as_error,omitempty"`
	// List of HTTP status codes to be excluded from being classified as an error. Error connections or responses impacts health score, are included as significant logs, and may be classified as part of a DoS attack. 
	ExcludeHttpErrorCodes []int64 `json:"exclude_http_error_codes,omitempty"`
	// Exclude dns queries to domains outside the domains configured in the DNS application profile from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeInvalidDnsDomainAsError bool `json:"exclude_invalid_dns_domain_as_error,omitempty"`
	// Exclude invalid dns queries from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeInvalidDnsQueryAsError bool `json:"exclude_invalid_dns_query_as_error,omitempty"`
	// Exclude the Issuer-Revoked OCSP Responses from the list of errors. Allowed in Basic(Allowed values- true) edition, Essentials(Allowed values- true) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	ExcludeIssuerRevokedOcspResponsesAsError bool `json:"exclude_issuer_revoked_ocsp_responses_as_error,omitempty"`
	// Exclude queries to domains that did not have configured services/records from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeNoDnsRecordAsError bool `json:"exclude_no_dns_record_as_error,omitempty"`
	// Exclude queries to GSLB services that have no available members from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeNoValidGsMemberAsError bool `json:"exclude_no_valid_gs_member_as_error,omitempty"`
	// Exclude persistence server changed while load balancing' from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludePersistenceChangeAsError bool `json:"exclude_persistence_change_as_error,omitempty"`
	// Exclude the Revoked OCSP certificate status responses from the list of errors. Allowed in Basic(Allowed values- true) edition, Essentials(Allowed values- true) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	ExcludeRevokedOcspResponsesAsError bool `json:"exclude_revoked_ocsp_responses_as_error,omitempty"`
	// Exclude server dns error response from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeServerDnsErrorAsError bool `json:"exclude_server_dns_error_as_error,omitempty"`
	// Exclude server TCP reset from errors. It is common for applications like MS Exchange. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeServerTcpResetAsError bool `json:"exclude_server_tcp_reset_as_error,omitempty"`
	// List of SIP status codes to be excluded from being classified as an error. Allowed in Basic edition, Essentials edition, Enterprise edition. 
	ExcludeSipErrorCodes []int64 `json:"exclude_sip_error_codes,omitempty"`
	// Exclude the Stale OCSP certificate status responses from the list of errors. Allowed in Basic(Allowed values- true) edition, Essentials(Allowed values- true) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	ExcludeStaleOcspResponsesAsError bool `json:"exclude_stale_ocsp_responses_as_error,omitempty"`
	// Exclude 'server unanswered syns' from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeSynRetransmitAsError bool `json:"exclude_syn_retransmit_as_error,omitempty"`
	// Exclude TCP resets by client from the list of potential errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeTcpResetAsError bool `json:"exclude_tcp_reset_as_error,omitempty"`
	// Exclude the unavailable OCSP Responses from the list of errors. Allowed in Basic(Allowed values- true) edition, Essentials(Allowed values- true) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as true. 
	ExcludeUnavailableOcspResponsesAsError bool `json:"exclude_unavailable_ocsp_responses_as_error,omitempty"`
	// Exclude unsupported dns queries from the list of errors. Allowed in Basic(Allowed values- false) edition, Essentials(Allowed values- false) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as false. 
	ExcludeUnsupportedDnsQueryAsError bool `json:"exclude_unsupported_dns_query_as_error,omitempty"`
	// Skips health score computation of pool servers when number of servers in a pool is more than this setting. Allowed values are 0-5000. Special values are 0- 'server health score is deactivated'. Allowed in Basic(Allowed values- 0) edition, Essentials(Allowed values- 0) edition, Enterprise edition. Special default for Basic edition is 0, Essentials edition is 0, Enterprise is 20. Default value when not specified in API or module is interpreted by ALB Controller as 0. 
	HealthscoreMaxServerLimit int64 `json:"healthscore_max_server_limit,omitempty"`
	// Time window (in secs) within which only unique health change events should occur. Allowed in Basic(Allowed values- 1209600) edition, Essentials(Allowed values- 1209600) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 1209600. 
	HsEventThrottleWindow int64 `json:"hs_event_throttle_window,omitempty"`
	// Maximum penalty that may be deducted from health score for anomalies. Allowed values are 0-100. Allowed in Basic(Allowed values- 10) edition, Essentials(Allowed values- 10) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 10. 
	HsMaxAnomalyPenalty int64 `json:"hs_max_anomaly_penalty,omitempty"`
	// Maximum penalty that may be deducted from health score for high resource utilization. Allowed values are 0-100. Allowed in Basic(Allowed values- 25) edition, Essentials(Allowed values- 25) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 25. 
	HsMaxResourcesPenalty int64 `json:"hs_max_resources_penalty,omitempty"`
	// Maximum penalty that may be deducted from health score based on security assessment. Allowed values are 0-100. Allowed in Basic(Allowed values- 100) edition, Essentials(Allowed values- 100) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 100. 
	HsMaxSecurityPenalty int64 `json:"hs_max_security_penalty,omitempty"`
	// DoS connection rate below which the DoS security assessment will not kick in. Allowed in Basic(Allowed values- 1000) edition, Essentials(Allowed values- 1000) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 1000. 
	HsMinDosRate int64 `json:"hs_min_dos_rate,omitempty"`
	// Adds free performance score credits to health score. It can be used for compensating health score for known slow applications. Allowed values are 0-100. Allowed in Basic(Allowed values- 0) edition, Essentials(Allowed values- 0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 0. 
	HsPerformanceBoost int64 `json:"hs_performance_boost,omitempty"`
	// Threshold number of connections in 5min, below which apdexr, apdexc, rum_apdex, and other network quality metrics are not computed. Allowed in Basic(Allowed values- 10) edition, Essentials(Allowed values- 10) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 10.0. 
	HsPscoreTrafficThresholdL4Client float32 `json:"hs_pscore_traffic_threshold_l4_client,omitempty"`
	// Threshold number of connections in 5min, below which apdexr, apdexc, rum_apdex, and other network quality metrics are not computed. Allowed in Basic(Allowed values- 10) edition, Essentials(Allowed values- 10) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 10.0. 
	HsPscoreTrafficThresholdL4Server float32 `json:"hs_pscore_traffic_threshold_l4_server,omitempty"`
	// Score assigned when the certificate has expired. Allowed values are 0-5. Allowed in Basic(Allowed values- 0.0) edition, Essentials(Allowed values- 0.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 0.0. 
	HsSecurityCertscoreExpired float32 `json:"hs_security_certscore_expired,omitempty"`
	// Score assigned when the certificate expires in more than 30 days. Allowed values are 0-5. Allowed in Basic(Allowed values- 5.0) edition, Essentials(Allowed values- 5.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 5.0. 
	HsSecurityCertscoreGt30d float32 `json:"hs_security_certscore_gt30d,omitempty"`
	// Score assigned when the certificate expires in less than or equal to 7 days. Allowed values are 0-5. Allowed in Basic(Allowed values- 2.0) edition, Essentials(Allowed values- 2.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 2.0. 
	HsSecurityCertscoreLe07d float32 `json:"hs_security_certscore_le07d,omitempty"`
	// Score assigned when the certificate expires in less than or equal to 30 days. Allowed values are 0-5. Allowed in Basic(Allowed values- 4.0) edition, Essentials(Allowed values- 4.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 4.0. 
	HsSecurityCertscoreLe30d float32 `json:"hs_security_certscore_le30d,omitempty"`
	// Penalty for allowing certificates with invalid chain. Allowed values are 0-5. Allowed in Basic(Allowed values- 1.0) edition, Essentials(Allowed values- 1.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 1.0. 
	HsSecurityChainInvalidityPenalty float32 `json:"hs_security_chain_invalidity_penalty,omitempty"`
	// Score assigned when the minimum cipher strength is 0 bits. Allowed values are 0-5. Allowed in Basic(Allowed values- 0.0) edition, Essentials(Allowed values- 0.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 0.0. 
	HsSecurityCipherscoreEq000b float32 `json:"hs_security_cipherscore_eq000b,omitempty"`
	// Score assigned when the minimum cipher strength is greater than equal to 128 bits. Allowed values are 0-5. Allowed in Basic(Allowed values- 5.0) edition, Essentials(Allowed values- 5.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 5.0. 
	HsSecurityCipherscoreGe128b float32 `json:"hs_security_cipherscore_ge128b,omitempty"`
	// Score assigned when the minimum cipher strength is less than 128 bits. Allowed values are 0-5. Allowed in Basic(Allowed values- 3.5) edition, Essentials(Allowed values- 3.5) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 3.5. 
	HsSecurityCipherscoreLt128b float32 `json:"hs_security_cipherscore_lt128b,omitempty"`
	// Score assigned when no algorithm is used for encryption. Allowed values are 0-5. Allowed in Basic(Allowed values- 0.0) edition, Essentials(Allowed values- 0.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 0.0. 
	HsSecurityEncalgoScoreNone float32 `json:"hs_security_encalgo_score_none,omitempty"`
	// Score assigned when RC4 algorithm is used for encryption. Allowed values are 0-5. Allowed in Basic(Allowed values- 2.5) edition, Essentials(Allowed values- 2.5) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 2.5. 
	HsSecurityEncalgoScoreRc4 float32 `json:"hs_security_encalgo_score_rc4,omitempty"`
	// Penalty for not enabling HSTS. Allowed values are 0-5. Allowed in Basic(Allowed values- 1.0) edition, Essentials(Allowed values- 1.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 1.0. 
	HsSecurityHstsPenalty float32 `json:"hs_security_hsts_penalty,omitempty"`
	// Penalty for allowing non-PFS handshakes. Allowed values are 0-5. Allowed in Basic(Allowed values- 1.0) edition, Essentials(Allowed values- 1.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 1.0. 
	HsSecurityNonpfsPenalty float32 `json:"hs_security_nonpfs_penalty,omitempty"`
	// Score assigned when OCSP Certificate Status is set to Revoked or Issuer Revoked. Allowed values are 0.0-5.0. Allowed in Basic(Allowed values- 0.0) edition, Essentials(Allowed values- 0.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 0.0. 
	HsSecurityOcspRevokedScore float32 `json:"hs_security_ocsp_revoked_score,omitempty"`
	// Deprecated. Allowed values are 0-5. Allowed in Basic(Allowed values- 1.0) edition, Essentials(Allowed values- 1.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 1.0. 
	HsSecuritySelfsignedcertPenalty float32 `json:"hs_security_selfsignedcert_penalty,omitempty"`
	// Score assigned when supporting SSL3.0 encryption protocol. Allowed values are 0-5. Allowed in Basic(Allowed values- 3.5) edition, Essentials(Allowed values- 3.5) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 3.5. 
	HsSecuritySsl30Score float32 `json:"hs_security_ssl30_score,omitempty"`
	// Score assigned when supporting TLS1.0 encryption protocol. Allowed values are 0-5. Allowed in Basic(Allowed values- 5.0) edition, Essentials(Allowed values- 5.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 5.0. 
	HsSecurityTls10Score float32 `json:"hs_security_tls10_score,omitempty"`
	// Score assigned when supporting TLS1.1 encryption protocol. Allowed values are 0-5. Allowed in Basic(Allowed values- 5.0) edition, Essentials(Allowed values- 5.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 5.0. 
	HsSecurityTls11Score float32 `json:"hs_security_tls11_score,omitempty"`
	// Score assigned when supporting TLS1.2 encryption protocol. Allowed values are 0-5. Allowed in Basic(Allowed values- 5.0) edition, Essentials(Allowed values- 5.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 5.0. 
	HsSecurityTls12Score float32 `json:"hs_security_tls12_score,omitempty"`
	// Score assigned when supporting TLS1.3 encryption protocol. Allowed values are 0-5. Allowed in Basic(Allowed values- 5.0) edition, Essentials(Allowed values- 5.0) edition, Enterprise edition. 
	HsSecurityTls13Score float32 `json:"hs_security_tls13_score,omitempty"`
	// Penalty for allowing weak signature algorithm(s). Allowed values are 0-5. Allowed in Basic(Allowed values- 1.0) edition, Essentials(Allowed values- 1.0) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 1.0. 
	HsSecurityWeakSignatureAlgoPenalty float32 `json:"hs_security_weak_signature_algo_penalty,omitempty"`
	// List of labels to be used for granular RBAC. Allowed in Basic edition, Essentials edition, Enterprise edition. 
	Markers []AlbRoleFilterMatchLabel `json:"markers,omitempty"`
	// This flag sets the time duration of no live data traffic after which Virtual Service metrics processing is suspended. It is applicable only when enable_ondemand_metrics is set to false. Unit is SECONDS. Default value when not specified in API or module is interpreted by ALB Controller as 1800. 
	OndemandMetricsIdleTimeout int64 `json:"ondemand_metrics_idle_timeout,omitempty"`
	// List of HTTP status code ranges to be excluded from being classified as an error. 
	Ranges []AlbhttpStatusRange `json:"ranges,omitempty"`
	// Block of HTTP response codes to be excluded from being classified as an error. Enum options - AP_HTTP_RSP_4XX, AP_HTTP_RSP_5XX. 
	RespCodeBlock []string `json:"resp_code_block,omitempty"`
	SensitiveLogProfile *AlbSensitiveLogProfile `json:"sensitive_log_profile,omitempty"`
	// Maximum number of SIP messages added in logs for a SIP transaction. By default, this value is 20. Allowed values are 1-1000. Allowed in Basic(Allowed values- 20) edition, Essentials(Allowed values- 20) edition, Enterprise edition. Default value when not specified in API or module is interpreted by ALB Controller as 20. 
	SipLogDepth int64 `json:"sip_log_depth,omitempty"`
}