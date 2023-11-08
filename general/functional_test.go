package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/deputy"
	"github.com/open-cyber-range/vmware-handler/grpc/event"
	"github.com/open-cyber-range/vmware-handler/library"
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
	ServerAddress:      "127.0.0.1",
	ResourcePoolPath:   os.Getenv("TEST_VMWARE_RESOURCE_POOL_PATH"),
	ExerciseRootPath:   os.Getenv("TEST_VMWARE_EXERCISE_ROOT_PATH"),
	RedisAddress:       os.Getenv("TEST_REDIS_ADDRESS"),
	RedisPassword:      os.Getenv("TEST_REDIS_PASSWORD"),
}

func startServer(timeout time.Duration) (configuration library.Configuration) {
	configuration = testConfiguration
	validator := library.NewValidator().SetRequireExerciseRootPath(true)
	err := configuration.Validate(validator)
	if err != nil {
		log.Fatalf("Failed to validate configuration: %v", err)
	}
	rand.Seed(time.Now().UnixNano())
	randomPort := rand.Intn(10000) + 10000
	configuration.ServerAddress = fmt.Sprintf("%v:%v", configuration.ServerAddress, randomPort)
	go RealMain(&configuration)

	time.Sleep(timeout)
	return configuration
}

func createEventClient(t *testing.T, serverPath string) event.EventInfoServiceClient {
	connection, connectionError := grpc.Dial(serverPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		t.Fatalf("Failed to connect to grpc server: %v", connectionError)
	}
	t.Cleanup(func() {
		connectionError := connection.Close()
		if connectionError != nil {
			t.Fatalf("Failed to close grpc connection: %v", connectionError)
		}
	})
	return event.NewEventInfoServiceClient(connection)
}

func createDeputyQueryClient(t *testing.T, serverPath string) deputy.DeputyQueryServiceClient {
	connection, connectionError := grpc.Dial(serverPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		t.Fatalf("Failed to connect to grpc server: %v", connectionError)
	}
	t.Cleanup(func() {
		connectionError := connection.Close()
		if connectionError != nil {
			t.Fatalf("Failed to close grpc connection: %v", connectionError)
		}
	})
	return deputy.NewDeputyQueryServiceClient(connection)
}

func sendEventDeleteRequest(t *testing.T, gRPCClient event.EventInfoServiceClient, identifier *common.Identifier) error {
	ctx := context.Background()
	_, err := gRPCClient.Delete(ctx, &common.Identifier{Value: identifier.GetValue()})
	if err != nil {
		return err
	}
	return nil
}

func createEventCreateRequest(t *testing.T, gRPCClient event.EventInfoServiceClient, eventRequest *common.Source, packageName string) (eventInfoResponse *event.EventCreateResponse, err error) {
	token := os.Getenv("TEST_DEPUTY_TOKEN")
	if err := library.PublishTestPackage(packageName, token); err != nil {
		t.Fatalf("Failed to upload test Event package: %v", err)
	}

	eventInfoResponse, err = gRPCClient.Create(context.Background(), eventRequest)
	if err != nil {
		t.Fatalf("Test Create request error: %v", err)
	}

	return
}

func TestCreateEventInfo(t *testing.T) {
	configuration := startServer(3 * time.Second)
	gRPCClient := createEventClient(t, configuration.ServerAddress)

	request := &common.Source{
		Name:    "handler-test-event-info",
		Version: "*",
	}
	eventInfoResponse, err := createEventCreateRequest(t, gRPCClient, request, "event-info-package")
	if err != nil {
		t.Fatalf("Failed to create event info: %v", err)
	}

	if err = sendEventDeleteRequest(t, gRPCClient, &common.Identifier{Value: eventInfoResponse.Id}); err != nil {
		t.Fatalf("Failed to delete event: %v", err)
	}

}

func TestStreamEventInfo(t *testing.T) {
	ctx := context.Background()
	configuration := startServer(3 * time.Second)
	gRPCClient := createEventClient(t, configuration.ServerAddress)

	request := &common.Source{
		Name:    "handler-test-event-info",
		Version: "*",
	}

	eventInfoResponse, err := createEventCreateRequest(t, gRPCClient, request, "event-info-package")
	if err != nil {
		t.Fatalf("Failed to create event info: %v", err)
	}

	stream, err := gRPCClient.Stream(ctx, &common.Identifier{Value: eventInfoResponse.Id})
	if err != nil {
		t.Fatalf("Test Stream request error: %v", err)
	}

	log.Infof("Received eventResponse: %v", eventInfoResponse)

	receivedFilePath := "/tmp" + "/" + eventInfoResponse.GetFilename()

	file, err := os.Create(receivedFilePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	for {
		chunkResponse, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("err receiving chunk: %v", err)
			break
		}
		file.Write(chunkResponse.Chunk)
	}

	receivedFileChecksum, err := library.GetMD5Checksum(receivedFilePath)
	if err != nil {
		t.Fatalf("Failed to get checksum of received file: %v", err)
	}

	if receivedFileChecksum != eventInfoResponse.Checksum {
		t.Fatalf("Received file checksum %v does not match expected checksum %v ", receivedFileChecksum, eventInfoResponse.Checksum)
	}

	if err = sendEventDeleteRequest(t, gRPCClient, &common.Identifier{Value: eventInfoResponse.Id}); err != nil {
		t.Fatalf("Failed to delete event: %v", err)
	}
}

func TestGetDeputyPackagesByType(t *testing.T) {
	ctx := context.Background()
	configuration := startServer(3 * time.Second)
	gRPCClient := createDeputyQueryClient(t, configuration.ServerAddress)

	token := os.Getenv("TEST_DEPUTY_TOKEN")
	if err := library.PublishTestPackage("condition-package", token); err != nil {
		t.Fatalf("Failed to upload test feature package: %v", err)
	}

	request := &deputy.GetPackagesQuery{PackageType: "condition"}
	response, err := gRPCClient.GetPackagesByType(ctx, request)
	if err != nil {
		t.Fatalf("Test GetPackagesByType request error: %v", err)
	}

	if len(response.Packages) == 0 {
		t.Fatalf("GetPackagesByType Received empty response")
	}

}

func TestGetScenario(t *testing.T) {
	ctx := context.Background()
	configuration := startServer(3 * time.Second)
	gRPCClient := createDeputyQueryClient(t, configuration.ServerAddress)

	request := &common.Source{
		Name:    "handler-test-exercise",
		Version: "*",
	}

	token := os.Getenv("TEST_DEPUTY_TOKEN")
	if err := library.PublishTestPackage(request.Name, token); err != nil {
		t.Fatalf("Failed to upload test Exercise package: %v", err)
	}

	response, err := gRPCClient.GetScenario(ctx, request)
	if err != nil {
		t.Fatalf("GetScenario request error: %v", err)
	}

	if response.Sdl == "" {
		t.Fatalf("GetScenario: Received empty response")
	}

}
