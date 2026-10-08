package v2

const (
	crmPBXPath           = "professional-crm-pbx-integration-call"
	crmPBXRoute          = "/v2/invoke_function/" + crmPBXPath
	crmPBXFunctionID     = "2b0f8e70-06b8-4fa9-98a4-aaea2a22c348"
	crmPBXIdentityMethod = "crm_pbx_invocation_identity"
	crmPBXOperatorRole   = "a2043881-f24e-4338-b558-99aa5eb200fb"
	crmPBXOperatorClient = "d3477291-2093-4a6e-a530-fd65097149fa"
	crmPBXAppPlatform    = "7d4a4c38-dd84-4902-b744-0488b80a4c04"
	crmPBXRequestLimit   = 16 << 10
	// Existing transcript storage accepts 512KiB decoded JSON; allow its outer
	// JSON escaping without enlarging limits for other business methods.
	crmPBXTranscriptRequestLimit = 6*(512<<10) + crmPBXRequestLimit
	crmPBXResponseLimit          = 2 << 20
)
