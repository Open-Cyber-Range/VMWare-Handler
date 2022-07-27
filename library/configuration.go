package library

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/vmware/govmomi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
)

type Validator struct {
	requireExerciseRootPath bool
	requireDatastorePath    bool
}

func NewValidator() *Validator {
	return &Validator{
		requireExerciseRootPath: false,
		requireDatastorePath:    false,
	}
}

func (validator *Validator) SetRequireExerciseRootPath(value bool) *Validator {
	validator.requireExerciseRootPath = value
	return validator
}

func (validator *Validator) SetRequireDatastorePath(value bool) *Validator {
	validator.requireDatastorePath = value
	return validator
}

func (validator *Validator) GetConfiguration() (configuration Configuration, err error) {
	commandArgs := os.Args
	if len(commandArgs) < 2 {
		return configuration, fmt.Errorf("no configuration path provided")
	}
	configurationPath := commandArgs[1]

	yamlFile, err := ioutil.ReadFile(configurationPath)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(yamlFile, &configuration)
	if err != nil {
		return
	}

	err = configuration.Validate(validator)
	return
}

type Configuration struct {
	User               string `yaml:",omitempty"`
	Password           string `yaml:",omitempty"`
	Hostname           string `yaml:",omitempty"`
	Insecure           bool   `yaml:",omitempty"`
	TemplateFolderPath string `yaml:"template_folder_path,omitempty"`
	ServerAddress      string `yaml:"server_address,omitempty"`
	ResourcePoolPath   string `yaml:"resource_pool_path,omitempty"`
	ExerciseRootPath   string `yaml:"exercise_root_path,omitempty"`
	DatastorePath      string `yaml:"datastore_path,omitempty"`
}

func (configuration *Configuration) Validate(validator *Validator) error {
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
	if configuration.ServerAddress == "" {
		return status.Error(codes.InvalidArgument, "Vsphere server address not provided")
	}
	if validator.requireExerciseRootPath && configuration.ExerciseRootPath == "" {
		return status.Error(codes.InvalidArgument, "Vsphere exercise root path not provided")
	}
	if validator.requireDatastorePath && configuration.DatastorePath == "" {
		return status.Error(codes.InvalidArgument, "Vsphere datastore path not provided")
	}
	return nil
}

func (configuration *Configuration) CreateClient(ctx context.Context) (*govmomi.Client, error) {
	hostURL, parseError := url.Parse("https://" + configuration.Hostname + "/sdk")

	if parseError != nil {
		return nil, fmt.Errorf("failed to parse url: %s", parseError)
	}

	hostURL.User = url.UserPassword(configuration.User, configuration.Password)
	client, clientError := govmomi.NewClient(ctx, hostURL, configuration.Insecure)

	if clientError != nil {
		return nil, fmt.Errorf("failed to setup the client: %s", clientError)
	}

	return client, nil
}
