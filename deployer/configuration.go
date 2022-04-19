package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/vmware/govmomi"
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	User               string
	Password           string
	Hostname           string
	Insecure           bool
	TemplateFolderPath string `yaml:"template_folder_path"`
	ResourcePoolPath   string `yaml:"resource_pool_path"`
	ExerciseRootPath   string `yaml:"exercise_root_path"`
	ServerAddress      string `yaml:"server_address"`
}

func (configuration *Configuration) createClient(ctx context.Context) (*govmomi.Client, error) {
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

func getConfiguration() (_ *Configuration, err error) {
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
