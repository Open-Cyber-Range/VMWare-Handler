package library

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
)

type Validator struct {
	requireExerciseRootPath     bool
	requireDatastorePath        bool
	requireVSphereConfiguration bool
	requireRedisConfiguration   bool
}

func NewValidator() *Validator {
	return &Validator{
		requireExerciseRootPath:     false,
		requireDatastorePath:        false,
		requireVSphereConfiguration: false,
		requireRedisConfiguration:   false,
	}
}

type ConfigurationVariables struct {
	MaxConnections          int64 `yaml:"max_connections,omitempty"`
	VmToolsTimeoutSec       int   `yaml:"vm_tools_timeout_sec,omitempty"`
	VmToolsRetrySec         int   `yaml:"vm_tools_retry_sec,omitempty"`
	VmPropertiesTimeoutSec  int   `yaml:"vm_properties_timeout_sec,omitempty"`
	MutexTimeoutSec         int   `yaml:"mutex_timeout_sec,omitempty"`
	MutexPoolMaxRetryMillis int   `yaml:"max_mutex_pool_retry_millis,omitempty"`
	MutexPoolMinRetryMillis int   `yaml:"min_mutex_pool_retry_millis,omitempty"`
	MutexLockMaxRetryMillis int   `yaml:"max_mutex_lock_retry_millis,omitempty"`
	MutexLockMinRetryMillis int   `yaml:"min_mutex_lock_retry_millis,omitempty"`
	ExecutorRunTimeoutSec   int   `yaml:"executor_run_timeout_sec,omitempty"`
	ExecutorRunRetrySec     int   `yaml:"executor_run_retry_sec,omitempty"`
}

type Configuration struct {
	User               string                 `yaml:",omitempty"`
	Password           string                 `yaml:",omitempty"`
	Hostname           string                 `yaml:",omitempty"`
	Insecure           bool                   `yaml:",omitempty"`
	TemplateFolderPath string                 `yaml:"template_folder_path,omitempty"`
	ServerAddress      string                 `yaml:"server_address,omitempty"`
	ResourcePoolPath   string                 `yaml:"resource_pool_path,omitempty"`
	ExerciseRootPath   string                 `yaml:"exercise_root_path,omitempty"`
	DatastorePath      string                 `yaml:"datastore_path,omitempty"`
	RedisAddress       string                 `yaml:"redis_address,omitempty"`
	RedisPassword      string                 `yaml:"redis_password,omitempty"`
	Variables          ConfigurationVariables `yaml:",inline"`
}

func (validator *Validator) SetRequireExerciseRootPath(value bool) *Validator {
	validator.requireExerciseRootPath = value
	return validator
}

func (validator *Validator) SetRequireDatastorePath(value bool) *Validator {
	validator.requireDatastorePath = value
	return validator
}

func (validator *Validator) SetRequireVSphereConfiguration(value bool) *Validator {
	validator.requireVSphereConfiguration = value
	return validator
}

func (validator *Validator) SetRequireRedisConfiguration(value bool) *Validator {
	validator.requireRedisConfiguration = value
	return validator
}

func (validator *Validator) GetConfiguration() (configuration Configuration, err error) {
	commandArgs := os.Args
	if len(commandArgs) < 2 {
		return configuration, fmt.Errorf("no configuration path provided")
	}
	configurationPath := commandArgs[1]

	yamlFile, err := os.ReadFile(configurationPath)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(yamlFile, &configuration)
	if err != nil {
		return
	}

	configuration.SetDefaultConfigurationValues()
	err = configuration.Validate(validator)
	return
}

func (configuration *Configuration) Validate(validator *Validator) error {
	if configuration.ServerAddress == "" {
		return status.Error(codes.InvalidArgument, "Handler address not provided")
	}

	if validator.requireVSphereConfiguration {
		if configuration.User == "" {
			return status.Error(codes.InvalidArgument, "Vsphere user name not provided")
		}
		if configuration.Password == "" {
			return status.Error(codes.InvalidArgument, "Vsphere password not provided")
		}
		if configuration.Hostname == "" {
			return status.Error(codes.InvalidArgument, "Vsphere host name not provided")
		}
		if configuration.TemplateFolderPath == "" {
			return status.Error(codes.InvalidArgument, "Vsphere template folder path not provided")
		}
		if configuration.ResourcePoolPath == "" {
			return status.Error(codes.InvalidArgument, "Vsphere resource pool path not provided")
		}
		if validator.requireExerciseRootPath && configuration.ExerciseRootPath == "" {
			return status.Error(codes.InvalidArgument, "Vsphere exercise root path not provided")
		}
		if validator.requireDatastorePath && configuration.DatastorePath == "" {
			return status.Error(codes.InvalidArgument, "Vsphere datastore path not provided")
		}

	}

	if validator.requireRedisConfiguration {
		if configuration.RedisAddress == "" {
			return status.Error(codes.InvalidArgument, "Redis server address not provided")
		}
		if configuration.RedisPassword == "" {
			return status.Error(codes.InvalidArgument, "Redis server password not provided")
		}
	}

	return nil
}

func (configuration *Configuration) SetDefaultConfigurationValues() {
	if configuration.Variables.MaxConnections == 0 {
		configuration.Variables.MaxConnections = DefaultConfigurationVariables.MaxConnections
	}
	if configuration.Variables.VmToolsTimeoutSec == 0 {
		configuration.Variables.VmToolsTimeoutSec = DefaultConfigurationVariables.VmToolsTimeoutSec
	}
	if configuration.Variables.VmToolsRetrySec == 0 {
		configuration.Variables.VmToolsRetrySec = DefaultConfigurationVariables.VmToolsRetrySec
	}
	if configuration.Variables.MutexTimeoutSec == 0 {
		configuration.Variables.MutexTimeoutSec = DefaultConfigurationVariables.MutexTimeoutSec
	}
	if configuration.Variables.MutexPoolMaxRetryMillis == 0 {
		configuration.Variables.MutexPoolMaxRetryMillis = DefaultConfigurationVariables.MutexPoolMaxRetryMillis
	}
	if configuration.Variables.MutexPoolMinRetryMillis == 0 {
		configuration.Variables.MutexPoolMinRetryMillis = DefaultConfigurationVariables.MutexPoolMinRetryMillis
	}
	if configuration.Variables.VmPropertiesTimeoutSec == 0 {
		configuration.Variables.VmPropertiesTimeoutSec = DefaultConfigurationVariables.VmPropertiesTimeoutSec
	}
	if configuration.Variables.MutexLockMaxRetryMillis == 0 {
		configuration.Variables.MutexLockMaxRetryMillis = DefaultConfigurationVariables.MutexLockMaxRetryMillis
	}
	if configuration.Variables.MutexLockMinRetryMillis == 0 {
		configuration.Variables.MutexLockMinRetryMillis = DefaultConfigurationVariables.MutexLockMinRetryMillis
	}
	if configuration.Variables.ExecutorRunTimeoutSec == 0 {
		configuration.Variables.ExecutorRunTimeoutSec = DefaultConfigurationVariables.ExecutorRunTimeoutSec
	}
	if configuration.Variables.ExecutorRunRetrySec == 0 {
		configuration.Variables.ExecutorRunRetrySec = DefaultConfigurationVariables.ExecutorRunRetrySec
	}
}

func isNotAuthenticated(err error) bool {
	if soap.IsSoapFault(err) {
		switch soap.ToSoapFault(err).VimFault().(type) {
		case types.NotAuthenticated:
			return true
		}
	}
	return false
}

func (configuration *Configuration) CreateClient(ctx context.Context) (*govmomi.Client, error) {
	hostURL, parseError := url.Parse("https://" + configuration.Hostname + "/sdk")

	if parseError != nil {
		return nil, fmt.Errorf("failed to parse url: %s", parseError)
	}

	hostURL.User = url.UserPassword(configuration.User, configuration.Password)
	soapClient := soap.NewClient(hostURL, configuration.Insecure)
	vimClient, vimClientError := vim25.NewClient(ctx, soapClient)
	if vimClientError != nil {
		return nil, fmt.Errorf("failed to create new client: %s", vimClientError)
	}

	sessionManager := session.NewManager(vimClient)
	client := &govmomi.Client{
		Client:         vimClient,
		SessionManager: sessionManager,
	}

	clientError := client.Login(ctx, hostURL.User)
	if clientError != nil {
		return nil, fmt.Errorf("failed to setup the client: %s", clientError)
	}

	vimClient.RoundTripper = session.KeepAliveHandler(vimClient, time.Duration(10)*time.Minute,
		func(roundTripper soap.RoundTripper) error {
			_, err := methods.GetCurrentTime(ctx, roundTripper)
			if err == nil {
				return nil
			}

			log.Warnf("session keepalive error: %s", err)

			if isNotAuthenticated(err) {

				if err = client.Login(ctx, hostURL.User); err != nil {
					log.Fatalf("session keepalive failed to re-authenticate: %s", err)
				} else {
					log.Info("session keepalive re-authenticated")
				}
			}

			return nil
		},
	)

	return client, nil
}
