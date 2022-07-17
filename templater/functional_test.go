package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/template"
	"github.com/open-cyber-range/vmware-handler/library"
	"github.com/phayes/freeport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var testConfiguration = library.VMWareConfiguration{
	User:               os.Getenv("TEST_VMWARE_USER"),
	Password:           os.Getenv("TEST_VMWARE_PASSWORD"),
	Hostname:           os.Getenv("TEST_VMWARE_HOSTNAME"),
	Insecure:           true,
	TemplateFolderPath: os.Getenv("TEST_VMWARE_TEMPLATE_FOLDER_PATH"),
	ServerAddress:      "127.0.0.1",
}

func startServer(timeout time.Duration) (configuration library.VMWareConfiguration, err error) {
	configuration = testConfiguration
	rand.Seed(time.Now().UnixNano())
	randomPort, err := freeport.GetFreePort()
	if err != nil {
		return
	}

	configuration.ServerAddress = fmt.Sprintf("%v:%v", configuration.ServerAddress, randomPort)
	go RealMain(configuration)

	time.Sleep(timeout)
	return configuration, nil
}

func creategRPCClient(t *testing.T, serverPath string) template.TemplateServiceClient {
	connection, connectionError := grpc.Dial(serverPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		t.Fatalf("did not connect: %v", connectionError)
	}
	t.Cleanup(func() {
		connectionError := connection.Close()
		if connectionError != nil {
			t.Fatalf("Failed to close connection: %v", connectionError)
		}
	})
	return template.NewTemplateServiceClient(connection)
}

func createTemplate(t *testing.T, client template.TemplateServiceClient) string {
	templateSource := &common.Source{
		Name:    "dsl-image",
		Version: "0.1.0",
	}

	creationResult, err := client.Create(context.Background(), templateSource)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	id := creationResult.GetValue()
	if id == "" {
		t.Logf("Failed to retrieve UUID")
	}
	return id
}

func TestTemplateCreation(t *testing.T) {
	t.Parallel()
	serverConfiguration, err := startServer(time.Second * 3)
	gRPCClient := creategRPCClient(t, serverConfiguration.ServerAddress)
	createTemplate(t, gRPCClient)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
}
