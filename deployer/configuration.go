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
	ServerPath         string `yaml:"server_path"`
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

func getConfiguration() (*Configuration, error) {
	var configuration Configuration
	commandArgs := os.Args
	if len(commandArgs) < 2 {
		return nil, fmt.Errorf("no configuration path provided")
	}
	configurationPath := commandArgs[1]

	yamlFile, fileError := ioutil.ReadFile(configurationPath)
	if fileError != nil {
		return nil, fileError
	}

	yamlParserError := yaml.Unmarshal(yamlFile, &configuration)
	if yamlParserError != nil {
		return nil, yamlParserError
	}

	return &configuration, nil
}
