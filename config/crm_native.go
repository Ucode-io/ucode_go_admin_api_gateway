package config

import "os"

type CRMNativeConfig struct {
	Enabled                                                  bool
	InboundEnabled                                           bool
	NativeWritersReady                                       bool
	Project, Environment, ResourceEnvironment, ServerScopeID string
}

func LoadCRMNative() CRMNativeConfig {
	return CRMNativeConfig{Enabled: os.Getenv("CRM_NATIVE_SCOPE_ENABLED") == "true", InboundEnabled: os.Getenv("CRM_INBOUND_ENABLED") == "true", NativeWritersReady: os.Getenv("CRM_NATIVE_WRITERS_READY") == "true", Project: os.Getenv("CRM_NATIVE_PROJECT_ID"), Environment: os.Getenv("CRM_NATIVE_ENVIRONMENT_ID"), ResourceEnvironment: os.Getenv("CRM_NATIVE_RESOURCE_ENVIRONMENT_ID"), ServerScopeID: os.Getenv("CRM_NATIVE_SERVER_SCOPE_ID")}
}
