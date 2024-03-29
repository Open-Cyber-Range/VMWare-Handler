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
	log "github.com/sirupsen/logrus"
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
}

func startServer(timeout time.Duration) (configuration library.Configuration, err error) {
	configuration = testConfiguration
	validator := library.NewValidator()
	validator.SetRequireDatastorePath(true)
	validator.SetRequireVSphereConfiguration(true)
	err = configuration.Validate(validator)
	if err != nil {
		log.Fatalf("Failed to validate configuration: %v", err)
	}
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

func createTemplate(t *testing.T, client template.TemplateServiceClient, templateSource *common.Source) string {

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
	vmwareClient, loginError := library.NewVMWareClient(ctx, govmomiClient, testConfiguration)
	if loginError != nil {
		t.Fatalf("Failed to login: %v", loginError)
	}
	serverConfiguration, err := startServer(time.Second * 3)
	gRPCClient := creategRPCClient(t, serverConfiguration.ServerAddress)

	templateSource := &common.Source{
		Name:    "alpine-minimal",
		Version: "*",
	}

	uuid := createTemplate(t, gRPCClient, templateSource)
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

func TestTemplateCreationWithNIC(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	govmomiClient, govmomiClientError := testConfiguration.CreateClient(ctx)
	if govmomiClientError != nil {
		t.Fatalf("Failed to create govmomi client: %v", govmomiClientError)
	}
	serverConfiguration, err := startServer(time.Second * 3)
	gRPCClient := creategRPCClient(t, serverConfiguration.ServerAddress)

	templateWithNicSource := &common.Source{
		Name:    "handler-nic-test",
		Version: "*",
	}
	vmwareClient, loginError := library.NewVMWareClient(ctx, govmomiClient, testConfiguration)
	if loginError != nil {
		t.Fatalf("Failed to login: %v", loginError)
	}

	uuid := createTemplate(t, gRPCClient, templateWithNicSource)
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
