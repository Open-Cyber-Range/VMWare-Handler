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
	ExecutorRunTimeoutSec:   300,
	ExecutorRunRetrySec:     2,
}

var PotentialRebootErrorList = []string{
	"guest operations agent could not be contacted",
	"operation cannot be performed in the current state",
	"operation is not allowed in the current state",
	"download(", // Toolbox Download api error
	"3016",      // vSphere api VIX error code
}
