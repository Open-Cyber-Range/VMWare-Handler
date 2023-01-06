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
	"github.com/open-cyber-range/vmware-handler/grpc/condition"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
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
	rand.Seed(time.Now().UnixNano())
	randomPort := rand.Intn(10000) + 10000
	configuration.ServerAddress = fmt.Sprintf("%v:%v", configuration.ServerAddress, randomPort)
	go RealMain(&configuration)

	time.Sleep(timeout)
	return configuration
}

func createFeatureClient(t *testing.T, serverPath string) feature.FeatureServiceClient {
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
	return feature.NewFeatureServiceClient(connection)
}

func createConditionClient(t *testing.T, serverPath string) condition.ConditionServiceClient {
	connection, connectionError := grpc.Dial(serverPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		t.Fatalf("did not connect: %v", connectionError)
	}

	t.Cleanup(func() {
		fmt.Println("Conditioner Cleanup")
		connectionError := connection.Close()
		if connectionError != nil {
			t.Fatalf("Failed to close connection: %v", connectionError)
		}
	})

	return condition.NewConditionServiceClient(connection)
}

func createFeatureDeploymentRequest(t *testing.T, deployment *feature.Feature, packageName string) (response *feature.FeatureResponse, err error) {
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	gRPCClient := createFeatureClient(t, configuration.ServerAddress)

	if err := library.PublishTestPackage(packageName); err != nil {
		t.Fatalf("Failed to upload test feature package: %v", err)
	}

	response, err = gRPCClient.Create(ctx, deployment)
	if err != nil {
		t.Fatalf("Test Create request error: %v", err)
	}

	log.Infof("Feature Create finished, id: %v", response.Identifier.GetValue())
	_, err = gRPCClient.Delete(ctx, response.Identifier)
	if err != nil {
		t.Fatalf("Test Delete request error: %v", err)
	}
	log.Infof("Feature delete finished")

	if deployment.FeatureType == feature.FeatureType_service {
		if response.VmLog == "" {
			t.Fatalf("Test Feature Service produced no logs and was likely not executed")
		}
	}
	return
}

func createConditionerDeploymentRequest(t *testing.T, deployment *condition.Condition) {
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	gRPCClient := createConditionClient(t, configuration.ServerAddress)

	identifier, err := gRPCClient.Create(ctx, deployment)
	if err != nil {
		t.Fatalf("Test Create request error: %v", err)
	}

	stream, err := gRPCClient.Stream(ctx, identifier)
	if err != nil {
		t.Fatalf("Test Stream request error: %v", err)
	}

	finished := make(chan bool)

	go func() {
		var responses int8
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				finished <- true
				return
			}
			if err != nil {
				log.Fatalf("Test Stream Receive error: %v", err)
			}

			responses += 1

			if responses == 3 {
				log.Printf("Received enough successful Condition responses, exiting")
				finished <- true
				return
			}

		}
	}()
	<-finished

	if deployment.Source != nil {
		_, err = gRPCClient.Delete(ctx, identifier)
		if err != nil {
			t.Fatalf("Test Delete request error: %v", err)
		}
		log.Infof("Condition Source deleted")
	}
}

func TestConditionerWithCommand(t *testing.T) {
	t.Parallel()

	deployment := &condition.Condition{
		Name:             "command-condition",
		VirtualMachineId: "4212b4a9-dd30-45cc-3667-b72c8dd97558",
		Account:          &common.Account{Username: "root", Password: "password"},
		Command:          "/prebaked-conditions/divider.sh",
		Interval:         1,
	}

	createConditionerDeploymentRequest(t, deployment)
}

func TestConditionerWithSourcePackage(t *testing.T) {
	t.Parallel()

	packageName := "condition-package"

	if err := library.PublishTestPackage(packageName); err != nil {
		t.Fatalf("Test publish failed: %v", err)
	}

	deployment := &condition.Condition{
		Name:             "source-condition",
		VirtualMachineId: "4212b4a9-dd30-45cc-3667-b72c8dd97558",
		Account:          &common.Account{Username: "root", Password: "password"},
		Source: &common.Source{
			Name:    "test-condition",
			Version: "*",
		},
	}

	createConditionerDeploymentRequest(t, deployment)
}

func TestFeatureServiceDeploymentAndDeletionOnLinux(t *testing.T) {
	t.Parallel()

	packageName := "feature-service-package"

	feature := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: "42127656-e390-d6a8-0703-c3425dbc8052",
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "test-service",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	response, err := createFeatureDeploymentRequest(t, feature, packageName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	log.Infof("Feature output: %#v", response.VmLog)

}

func TestFeatureConfigurationDeploymentAndDeletionOnLinux(t *testing.T) {
	t.Parallel()

	packageName := "feature-config-package"

	feature := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: "42127656-e390-d6a8-0703-c3425dbc8052",
		FeatureType:      feature.FeatureType_configuration,
		Source: &common.Source{
			Name:    "test-configuration",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	_, err := createFeatureDeploymentRequest(t, feature, packageName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}
}

func TestFeatureServiceDeploymentAndDeletionOnWindows(t *testing.T) {
	t.Parallel()
	
	packageName := "feature-win-service-package"

	feature := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: "42122b12-3a17-c0fb-eb3c-7cd935bb595b",
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "test-windows-service",
			Version: "*",
		},
		Account: &common.Account{Username: "user", Password: "password"},
	}
	response, err := createFeatureDeploymentRequest(t, feature, packageName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	log.Infof("Feature output: %#v", response.VmLog)
}
