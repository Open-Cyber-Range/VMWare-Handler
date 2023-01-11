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

var testConfiguration = library.Configuration{
	User:               os.Getenv("TEST_VMWARE_USER"),
	Password:           os.Getenv("TEST_VMWARE_PASSWORD"),
	Hostname:           os.Getenv("TEST_VMWARE_HOSTNAME"),
	Insecure:           true,
	TemplateFolderPath: os.Getenv("TEST_VMWARE_TEMPLATE_FOLDER_PATH"),
	ResourcePoolPath:   os.Getenv("TEST_VMWARE_RESOURCE_POOL_PATH"),
	DatastorePath:      os.Getenv("TEST_VMWARE_DATASTORE_PATH"),
	ServerAddress:      "127.0.0.1",
	RedisAddress:       os.Getenv("TEST_REDIS_ADDRESS"),
	RedisPassword:      os.Getenv("TEST_REDIS_PASSWORD"),
}

func startServer(timeout time.Duration) (configuration library.Configuration, err error) {
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

	uploadError := library.PublishTestPackage("small-ova-package")
	if uploadError != nil {
		t.Fatalf("Failed to upload deputy package: %v", uploadError)
	}
	templateSource := &common.Source{
		Name:    "dsl-image",
		Version: "*",
	}

	creationResult, err := client.Create(context.Background(), templateSource)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	id := creationResult.GetIdentifier().GetValue()
	if id == "" {
		t.Logf("Failed to retrieve UUID")
	}
	return id
}

func TestTemplateCreation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	govmomiClient, govmomiClientError := testConfiguration.CreateClient(ctx)
	if govmomiClientError != nil {
		t.Fatalf("Failed to create govmomi client: %v", govmomiClientError)
	}
	vmwareClient := library.NewVMWareClient(govmomiClient, testConfiguration.TemplateFolderPath)
	serverConfiguration, err := startServer(time.Second * 3)
	gRPCClient := creategRPCClient(t, serverConfiguration.ServerAddress)
	uuid := createTemplate(t, gRPCClient)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	t.Cleanup(func() {
		cleanupError := vmwareClient.DeleteVirtualMachineByUUID(uuid)
		if cleanupError != nil {
			t.Fatalf("Failed to cleanup: %v", cleanupError)
		}
	})
}
