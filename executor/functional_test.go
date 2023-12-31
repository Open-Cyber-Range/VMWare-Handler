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
	"github.com/open-cyber-range/vmware-handler/grpc/inject"
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

const LinuxTestVirtualMachineUUID = "4212b4a9-dd30-45cc-3667-b72c8dd97558"
const LinuxConditionsTestVirtualMachineUUID = "42128ddc-dfda-786d-905f-3d4db40b7cbb"
const WindowsTestVirtualMachineUUID = "42122b12-3a17-c0fb-eb3c-7cd935bb595b"
const WindowsConditionsTestVirtualMachineUUID = "4212d188-36c8-d2c3-c14d-98f2c25434c9"

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

func createFeatureClient(t *testing.T, serverPath string) feature.FeatureServiceClient {
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
	return feature.NewFeatureServiceClient(connection)
}

func createConditionClient(t *testing.T, serverPath string) condition.ConditionServiceClient {
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

	return condition.NewConditionServiceClient(connection)
}

func createInjectClient(t *testing.T, serverPath string) inject.InjectServiceClient {
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
	return inject.NewInjectServiceClient(connection)
}

func createFeatureDeploymentRequest(t *testing.T, deployment *feature.Feature, packageName string) (executorResponse *common.ExecutorResponse, err error) {
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	gRPCClient := createFeatureClient(t, configuration.ServerAddress)

	if err := library.PublishTestPackage(packageName); err != nil {
		t.Fatalf("Failed to upload test feature package: %v", err)
	}

	executorResponse, err = gRPCClient.Create(ctx, deployment)
	if err != nil {
		t.Fatalf("Test Create request error: %v", err)
	}

	log.Infof("Feature Create finished, id: %v", executorResponse.Identifier.GetValue())
	_, err = gRPCClient.Delete(ctx, executorResponse.Identifier)
	if err != nil {
		t.Fatalf("Test Delete request error: %v", err)
	}
	log.Infof("Feature delete finished")

	if deployment.FeatureType == feature.FeatureType_service {
		if executorResponse.VmLog == "" {
			t.Fatalf("Test Feature Service produced no logs and was likely not executed")
		}
	}
	return
}

func createInjectDeploymentRequest(t *testing.T, deployment *inject.Inject, packageName string) (executorResponse *common.ExecutorResponse, err error) {
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	gRPCClient := createInjectClient(t, configuration.ServerAddress)

	if err := library.PublishTestPackage(packageName); err != nil {
		t.Fatalf("Failed to upload test inject package: %v", err)
	}

	executorResponse, err = gRPCClient.Create(ctx, deployment)
	if err != nil {
		t.Fatalf("Test Create request error: %v", err)
	}

	log.Infof("Inject Create finished, id: %v", executorResponse.Identifier.GetValue())
	_, err = gRPCClient.Delete(ctx, executorResponse.Identifier)
	if err != nil {
		t.Fatalf("Test Delete request error: %v", err)
	}
	log.Infof("Inject delete finished")

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
			result, err := stream.Recv()
			if err == io.EOF {
				log.Fatalf("Test Stream EOF error: %v", err)
				finished <- true
				return
			}
			if err != nil {
				log.Fatalf("Test Stream Receive error: %v", err)
			}

			responses += 1
			log.Infof("Condition stream: %v received: %v\n", deployment.Name, result.CommandReturnValue)
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

	deployment := &condition.Condition{
		Name:             "command-condition",
		VirtualMachineId: LinuxConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "root", Password: "password"},
		Command:          "/prebaked-conditions/divider.sh",
		Interval:         5,
	}

	createConditionerDeploymentRequest(t, deployment)
}

func TestConditionerWithCommandAndEnvironment(t *testing.T) {

	deployment := &condition.Condition{
		Name:             "command-condition",
		VirtualMachineId: LinuxConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "root", Password: "password"},
		Command:          "echo $TEST1",
		Interval:         5,
		Environment:      []string{"TEST1=1"},
	}

	createConditionerDeploymentRequest(t, deployment)
}

func TestConditionerWithSourcePackage(t *testing.T) {

	packageFolderName := "condition-package"

	if err := library.PublishTestPackage(packageFolderName); err != nil {
		t.Fatalf("Test publish failed: %v", err)
	}

	deployment := &condition.Condition{
		Name:             "source-condition",
		VirtualMachineId: LinuxConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "root", Password: "password"},
		Source: &common.Source{
			Name:    "test-condition",
			Version: "*",
		},
	}

	createConditionerDeploymentRequest(t, deployment)
}

func TestConditionerWithSourcePackageOnWindows(t *testing.T) {

	packageFolderName := "condition-win-package"

	if err := library.PublishTestPackage(packageFolderName); err != nil {
		t.Fatalf("Test publish failed: %v", err)
	}

	deployment := &condition.Condition{
		Name:             "source-condition",
		VirtualMachineId: WindowsConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "user", Password: "password"},
		Source: &common.Source{
			Name:    "test-windows-condition",
			Version: "*",
		},
	}

	createConditionerDeploymentRequest(t, deployment)
}

func TestFeatureServiceDeploymentAndDeletionOnLinux(t *testing.T) {

	packageFolderName := "feature-service-package"

	deployment := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "test-service",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	response, err := createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	log.Infof("Feature output: %#v", response.VmLog)

}

func TestFeaturePackageWithALotOfFiles(t *testing.T) {

	packageFolderName := "feature-plethora"

	deployment := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_configuration,
		Source: &common.Source{
			Name:    "feature-plethora",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}
}

func TestFeatureConfigurationDeploymentAndDeletionOnLinux(t *testing.T) {

	packageFolderName := "feature-config-package"

	deployment := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_configuration,
		Source: &common.Source{
			Name:    "test-configuration",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}
}

func TestFeatureServiceDeploymentAndDeletionOnWindows(t *testing.T) {

	packageFolderName := "feature-win-service-package"

	deployment := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: WindowsTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "test-windows-service",
			Version: "*",
		},
		Account: &common.Account{Username: "user", Password: "password"},
	}
	response, err := createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	log.Infof("Feature output: %v", response.VmLog)
}

func TestInjectDeploymentAndDeletionOnLinux(t *testing.T) {

	packageFolderName := "inject-flag-generator"

	deployment := &inject.Inject{
		Name:             "test-inject",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		Source: &common.Source{
			Name:    "flag-generator",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	response, err := createInjectDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating Test Inject Deployment: %v", err)
	}
	log.Infof("Inject output: %v", response.VmLog)
}
