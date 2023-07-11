package library

const TmpPackagePrefix string = "deputy-package"

var DefaultConfigurationVariables = ConfigurationVariables{
	MaxConnections:          50,
	VmToolsTimeoutSec:       600,
	VmToolsRetrySec:         5,
	MutexTimeoutSec:         360,
	VmPropertiesTimeoutSec:  60,
	MutexPoolMaxRetryMillis: 50,
	MutexPoolMinRetryMillis: 20,
	MutexLockMaxRetryMillis: 100,
	MutexLockMinRetryMillis: 50,
}
