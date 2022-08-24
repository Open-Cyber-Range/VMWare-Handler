package main

import (
	"crypto/tls"
	"fmt"
	swagger "github.com/open-cyber-range/vmware-handler/switcher/yolo-go"
	"io/ioutil"
	"net/http"
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

func CreateAPIConfiguration(serverConfiguration *Configuration) (apiConfiguration *swagger.Configuration) {
	apiConfiguration = swagger.NewConfiguration()
	apiConfiguration.BasePath = "https://" + serverConfiguration.NsxtApi + "/policy/api/v1"
	apiConfiguration.DefaultHeader["Authorization"] = fmt.Sprintf("Basic %v", serverConfiguration.NsxtAuth)
	apiConfiguration.HTTPClient = &http.Client{
		// TODO TLS still insecure
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: serverConfiguration.Insecure},
		},
	}
	return
}
