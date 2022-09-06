package library

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/session/keepalive"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
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
	NsxtApi			   string `yaml:"nsxt_api,omitempty"`
	NsxtAuth           string `yaml:"nsxt_auth,omitempty"`
	TransportZoneName  string `yaml:"transport_zone_name,omitempty"`
	SiteId             string `yaml:"site_id,omitempty"`
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
	if configuration.NsxtApi == "" {
		return status.Error(codes.InvalidArgument, "NSX-T API not provided")
	}
	if configuration.NsxtAuth == "" {
		return status.Error(codes.InvalidArgument, "NSX-T  Authorization key not provided")
	}
	if configuration.TransportZoneName == "" {
		return status.Error(codes.InvalidArgument, "NSX-T  Transport Zone Name not provided")
	}
	if configuration.SiteId == "" {
		configuration.SiteId = "default"
	}
	return nil
}

func (configuration *Configuration) CreateClient(ctx context.Context) (*govmomi.Client, error) {
	hostURL, parseError := url.Parse("https://" + configuration.Hostname + "/sdk")

	if parseError != nil {
		return nil, fmt.Errorf("failed to parse url: %s", parseError)
	}

	hostURL.User = url.UserPassword(configuration.User, configuration.Password)
	soapClient := soap.NewClient(hostURL, configuration.Insecure)
	vimClient, _ := vim25.NewClient(ctx, soapClient)
	vimClient.RoundTripper = keepalive.NewHandlerSOAP(vimClient.RoundTripper, time.Duration(10)*time.Minute, nil)
	sessionManager := session.NewManager(vimClient)
	client := &govmomi.Client{
		Client:         vimClient,
		SessionManager: sessionManager,
	}

	clientError := client.Login(ctx, hostURL.User)
	if clientError != nil {
		return nil, fmt.Errorf("failed to setup the client: %s", clientError)
	}

	return client, nil
}
