package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/open-cyber-range/vmware-handler/library"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	library.VMWareConfiguration
	ExerciseRootPath string `yaml:"exercise_root_path,omitempty"`
}

func (configuration *Configuration) Validate() error {
	configuration.ValidateVMWare()
	if configuration.ExerciseRootPath == "" {
		return status.Error(codes.InvalidArgument, "Vsphere exercise root path not provided")
	}
	return nil
}

func GetConfiguration() (configuration Configuration, err error) {
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

	err = configuration.Validate()
	return configuration, err
}
