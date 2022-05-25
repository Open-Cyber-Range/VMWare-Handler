package deployer

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

type Configuration struct {
	User               string `yaml:",omitempty"`
	Password           string `yaml:",omitempty"`
	Hostname           string `yaml:",omitempty"`
	Insecure           bool   `yaml:",omitempty"`
	TemplateFolderPath string `yaml:"template_folder_path,omitempty"`
	ResourcePoolPath   string `yaml:"resource_pool_path,omitempty"`
	ExerciseRootPath   string `yaml:"exercise_root_path,omitempty"`
	ServerAddress      string `yaml:"server_address,omitempty"`
}

func (configuration *Configuration) Validate() error {
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
	if configuration.ExerciseRootPath == "" {
		return status.Error(codes.InvalidArgument, "Vsphere exercise root path not provided")
	}
	if configuration.ServerAddress == "" {
		return status.Error(codes.InvalidArgument, "Vsphere server address not provided")
	}
	return nil
}

func (configuration *Configuration) createClient(ctx context.Context) (*govmomi.Client, error) {
	validationError := configuration.Validate()
	if validationError != nil {
		return nil, validationError
	}
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

func GetConfiguration() (_ *Configuration, err error) {
	var configuration Configuration
	commandArgs := os.Args
	if len(commandArgs) < 2 {
		return nil, fmt.Errorf("no configuration path provided")
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

	return &configuration, nil
}
