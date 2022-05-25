package switcher

import (
	"fmt"
	"io/ioutil"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	ServerAddress     string `yaml:"server_address,omitempty"`
	NsxtApi           string `yaml:"nsxt_api,omitempty"`
	NsxtAuth          string `yaml:"nsxt_auth,omitempty"`
	TransportZoneName string `yaml:"transport_zone_name,omitempty"`
	Insecure          bool   `yaml:",omitempty"`
}

func (serverConfiguration *Configuration) Validate() error {
	if serverConfiguration.NsxtApi == "" {
		return status.Error(codes.InvalidArgument, "NSX-T API not provided")
	}
	if serverConfiguration.NsxtAuth == "" {
		return status.Error(codes.InvalidArgument, "NSX-T  Authorization key not provided")
	}
	if serverConfiguration.TransportZoneName == "" {
		return status.Error(codes.InvalidArgument, "NSX-T  Transport Zone Name not provided")
	}
	return nil
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
