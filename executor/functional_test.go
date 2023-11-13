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
	"github.com/vmware/govmomi/vim25/types"
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

const LinuxTestVirtualMachineUUID = "422df4b2-e516-6f4a-3707-f4a379553a5a"
const LinuxConditionsTestVirtualMachineUUID = "422d914a-6e37-5557-885e-ab92e1fc830b"
const WindowsTestVirtualMachineUUID = "422d31fd-d355-2133-8a62-829e034b1095"
const WindowsConditionsTestVirtualMachineUUID = "422d4097-5852-be8d-3be7-70e3483996e7"

func startServer(timeout time.Duration) (configuration library.Configuration) {
	configuration = testConfiguration
	configuration.SetDefaultConfigurationValues()
	validator := library.NewValidator()
	validator.SetRequireVSphereConfiguration(true)
	validator.SetRequireExerciseRootPath(true)
	validator.SetRequireRedisConfiguration(true)
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
func rebootVirtualMachine(t *testing.T, virtualMachineUUID string, configuration library.Configuration, vmAuthentication types.NamePasswordAuthentication) {
	ctx := context.Background()
	govmomiClient, clientError := configuration.CreateClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	vmwareClient := library.NewVMWareClient(govmomiClient, configuration.TemplateFolderPath, configuration.Variables)
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, virtualMachineUUID, &vmAuthentication)
	if err != nil {
		log.Fatalf("Failed to create guest managers, %v", err)
	}

	err = guestManager.Reboot(ctx)
	if err != nil {
		log.Fatalf("Failed to reboot %v", err)
	}
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
	token := os.Getenv("TEST_DEPUTY_TOKEN")
	if err := library.PublishTestPackage(packageName, token); err != nil {
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
	token := os.Getenv("TEST_DEPUTY_TOKEN")
	if err := library.PublishTestPackage(packageName, token); err != nil {
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

func createConditionerDeploymentRequest(t *testing.T, deployment *condition.Condition, rebootFlag bool) {
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	gRPCClient := createConditionClient(t, configuration.ServerAddress)

	log.Infof("Creating Deployment for Condition: %v", deployment)
	identifier, err := gRPCClient.Create(ctx, deployment)
	if err != nil {
		t.Fatalf("Test Create request error: %v", err)
	}

	stream, err := gRPCClient.Stream(ctx, identifier)
	if err != nil {
		t.Fatalf("Test Stream request error: %v", err)
	}

	vmAuthentication := types.NamePasswordAuthentication{
		Username: deployment.Account.Username,
		Password: deployment.Account.Password,
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

			if rebootFlag && responses > 1 {
				rebootFlag = false
				log.Infof("Rebooting test machine: %v", deployment.VirtualMachineId)
				rebootVirtualMachine(t, deployment.VirtualMachineId, configuration, vmAuthentication)
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

	createConditionerDeploymentRequest(t, deployment, false)
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

	createConditionerDeploymentRequest(t, deployment, false)
}

func TestConditionerWithSourcePackage(t *testing.T) {

	packageFolderName := "condition-package"
	token := os.Getenv("TEST_DEPUTY_TOKEN")

	if err := library.PublishTestPackage(packageFolderName, token); err != nil {
		t.Fatalf("Test publish failed: %v", err)
	}

	deployment := &condition.Condition{
		Name:             "source-condition",
		VirtualMachineId: LinuxConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "root", Password: "password"},
		Source: &common.Source{
			Name:    "handler-test-condition",
			Version: "*",
		},
	}

	createConditionerDeploymentRequest(t, deployment, false)
}

func TestConditionerWithSourcePackageOnWindows(t *testing.T) {

	packageFolderName := "condition-win-package"
	token := os.Getenv("TEST_DEPUTY_TOKEN")

	if err := library.PublishTestPackage(packageFolderName, token); err != nil {
		t.Fatalf("Test publish failed: %v", err)
	}

	deployment := &condition.Condition{
		Name:             "source-condition",
		VirtualMachineId: WindowsConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "test-user", Password: "Passw0rd"},
		Source: &common.Source{
			Name:    "handler-test-windows-condition",
			Version: "*",
		},
	}

	createConditionerDeploymentRequest(t, deployment, false)
}

func TestFeatureServiceDeploymentAndDeletionOnLinux(t *testing.T) {

	packageFolderName := "feature-service-package"

	deployment := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "handler-test-service",
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
			Name:    "handler-test-feature-plethora",
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
			Name:    "handler-test-configuration",
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
			Name:    "handler-test-windows-service",
			Version: "*",
		},
		Account: &common.Account{Username: "test-user", Password: "Passw0rd"},
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
			Name:    "handler-flag-generator",
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

func TestRebootingFeatureOnLinux(t *testing.T) {

	packageFolderName := "feature-restarting-service"

	deployment := &feature.Feature{
		Name:             "handler-test-restarting-service",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "handler-test-restarting-service",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	packageFolderName = "feature-config-package"

	deployment = &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_configuration,
		Source: &common.Source{
			Name:    "handler-test-configuration",
			Version: "*",
		},
		Account: &common.Account{Username: "root", Password: "password"},
	}
	_, err = createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating second Test Feature Deployment after reboot: %v", err)
	}

}

func TestRebootingFeatureOnWindows(t *testing.T) {

	packageFolderName := "feature-win-restarting-service"

	deployment := &feature.Feature{
		Name:             "handler-test-restarting-windows-service",
		VirtualMachineId: WindowsTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "handler-test-restarting-windows-service",
			Version: "*",
		},
		Account: &common.Account{Username: "test-user", Password: "Passw0rd"},
	}
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	packageFolderName = "feature-win-service-package"

	deployment = &feature.Feature{
		Name:             "handler-test-windows-service",
		VirtualMachineId: WindowsTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_configuration,
		Source: &common.Source{
			Name:    "handler-test-windows-service",
			Version: "*",
		},
		Account: &common.Account{Username: "test-user", Password: "Passw0rd"},
	}
	_, err = createFeatureDeploymentRequest(t, deployment, packageFolderName)
	if err != nil {
		t.Fatalf("Error creating second Test Feature Deployment after reboot: %v", err)
	}

}

func TestConditionRebootWhileStreaming(t *testing.T) {

	packageFolderName := "condition-package"
	token := os.Getenv("TEST_DEPUTY_TOKEN")

	if err := library.PublishTestPackage(packageFolderName, token); err != nil {
		t.Fatalf("Test publish failed: %v", err)
	}

	deployment := &condition.Condition{
		Name:             "source-condition",
		VirtualMachineId: LinuxConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "root", Password: "password"},
		Source: &common.Source{
			Name:    "handler-test-condition",
			Version: "*",
		},
	}

	createConditionerDeploymentRequest(t, deployment, true)
}

func TestConditionRebootWhileStreamingOnWindows(t *testing.T) {

	packageFolderName := "condition-win-package"
	token := os.Getenv("TEST_DEPUTY_TOKEN")

	if err := library.PublishTestPackage(packageFolderName, token); err != nil {
		t.Fatalf("Test publish failed: %v", err)
	}

	deployment := &condition.Condition{
		Name:             "source-condition",
		VirtualMachineId: WindowsConditionsTestVirtualMachineUUID,
		Account:          &common.Account{Username: "test-user", Password: "Passw0rd"},
		Source: &common.Source{
			Name:    "handler-test-windows-condition",
			Version: "*",
		},
	}

	createConditionerDeploymentRequest(t, deployment, true)
}
