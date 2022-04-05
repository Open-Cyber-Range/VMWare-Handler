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
	TemplateFolderPath string
	ResourcePoolPath   string
	ExerciseRootPath   string
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
	configurationPath := os.Args[1]
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
