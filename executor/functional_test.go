package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	goredislib "github.com/go-redis/redis/v8"
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

	vmwareClient, loginError := library.NewVMWareClient(ctx, govmomiClient, configuration)
	if loginError != nil {
		log.Fatal(loginError)
	}
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

func sendFeatureDeleteMessage(t *testing.T, gRPCClient feature.FeatureServiceClient, identifier *common.Identifier) (err error) {
	ctx := context.Background()
	_, err = gRPCClient.Delete(ctx, identifier)
	if err != nil {
		return err
	}

	return
}

func createFeatureDeleteRequest(t *testing.T, identifier *common.Identifier) (err error) {
	configuration := startServer(3 * time.Second)
	gRPCClient := createFeatureClient(t, configuration.ServerAddress)
	err = sendFeatureDeleteMessage(t, gRPCClient, identifier)

	return
}

func createFeatureDeploymentRequest(t *testing.T, deployment *feature.Feature, packageName string, cleanup bool) (executorResponse *common.ExecutorResponse, err error) {
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

	if cleanup {
		if err = sendFeatureDeleteMessage(t, gRPCClient, executorResponse.Identifier); err != nil {
			t.Fatalf("Test Delete request error: %v", err)
		}
		log.Infof("Feature delete finished")
		if err = cleanupRedis(); err != nil {
			t.Fatalf("Test Redis cleanup error: %v", err)
		}

	}

	if deployment.FeatureType == feature.FeatureType_service {
		if executorResponse.Stdout == "" && executorResponse.Stderr == "" {
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
	err = cleanupRedis()

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

	var responses int8
	for {
		result, err := stream.Recv()
		if err == io.EOF {
			log.Errorf("Test Stream EOF error: %v", err)
			break
		}
		if err != nil {
			t.Fatalf("Test Stream Receive error: %v", err)
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
			if closeErr := stream.CloseSend(); closeErr != nil {
				log.Errorf("Error closing stream: %v", closeErr)
			}
			break
		}
	}

	if deployment.Source != nil {
		_, err = gRPCClient.Delete(ctx, identifier)
		if err != nil {
			t.Fatalf("Test Delete request error: %v", err)
		}
		log.Infof("Condition Source deleted")
	}
	if err = cleanupRedis(); err != nil {
		log.Fatalf("Test Redis cleanup error: %v", err)
	}
}

func getAssetsListFromPackage(t *testing.T, packageFolderName string) (assets [][]string) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return
	}
	packagePath := path.Join(workingDirectory, "..", "extra", "test-deputy-packages", packageFolderName)
	packageData, err := library.GetPackageData(packagePath)
	if err != nil {
		return
	}

	infoJson, err := json.Marshal(&packageData)
	if err != nil {
		t.Fatalf("Error marshalling Toml contents: %v", err)
	}
	executorPackage := library.ExecutorPackage{}
	if err = json.Unmarshal(infoJson, &executorPackage); err != nil {
		t.Fatalf("Error unmarshalling Toml contents: %v", err)
	}

	return executorPackage.PackageBody.Assets
}

func cleanupRedis() error {
	ctx := context.Background()
	configuration := testConfiguration
	redisClient := goredislib.NewClient(&goredislib.Options{
		Addr:     configuration.RedisAddress,
		Password: configuration.RedisPassword,
	})

	status, err := redisClient.FlushDB(ctx).Result()
	log.Infof("Redis flush status: %v", status)
	return err
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
	response, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	log.Infof("Feature stdout: %#v. Stderr: %#v", response.Stdout, response.Stderr)

}

func TestFeatureServiceDeploymentWithEnvironmentOnLinux(t *testing.T) {

	packageFolderName := "feature-service-package"

	deployment := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "environment-test",
			Version: "*",
		},
		Account:     &common.Account{Username: "root", Password: "password"},
		Environment: []string{"TEST_ENV1=hello", "TEST_ENV2=world"},
	}
	response, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	log.Infof("Feature stdout: %#v. Stderr: %#v", response.Stdout, response.Stderr)

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
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
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
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
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
	response, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}

	log.Infof("Feature stdout: %#v. Stderr: %#v", response.Stdout, response.Stderr)
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
	log.Infof("Inject stdout: %#v. Stderr: %#v", response.Stdout, response.Stderr)
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
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
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
	_, err = createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
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
	_, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
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
	_, err = createFeatureDeploymentRequest(t, deployment, packageFolderName, true)
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

func getDeployedAssetPermissions(t *testing.T, virtualMachineUUID string, vmAuthentication types.NamePasswordAuthentication, assets [][]string) (deployedFilePermissions []string) {
	ctx := context.Background()
	configuration := testConfiguration
	govmomiClient, clientError := configuration.CreateClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	vmwareClient, loginError := library.NewVMWareClient(ctx, govmomiClient, configuration)
	if loginError != nil {
		log.Fatal(loginError)
	}
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, virtualMachineUUID, &vmAuthentication)
	if err != nil {
		log.Fatalf("Failed to create guest managers, %v", err)
	}

	for _, asset := range assets {
		sourcePath := asset[0]
		destinationPath := asset[1]

		if strings.HasSuffix(destinationPath, "/") {
			destinationPath = strings.Join([]string{destinationPath, path.Base(sourcePath)}, "")
		}

		spec := []string{"stat", "-c", "'%a'", destinationPath}
		stdoutBuffer := new(bytes.Buffer)
		stderrBuffer := new(bytes.Buffer)

		cmd := &exec.Cmd{
			Path:   spec[0],
			Stdout: stdoutBuffer,
			Stderr: stderrBuffer,
			Args:   spec[1:],
		}

		if err = guestManager.Toolbox.Run(ctx, cmd); err != nil || stderrBuffer.Len() > 0 {
			log.Fatalf("Failed to run command. Err: %v, StdErr: %v", err, stderrBuffer.String())
		}

		deployedFilePermission := strings.TrimSpace(stdoutBuffer.String())
		deployedFilePermissions = append(deployedFilePermissions, deployedFilePermission)

	}

	if err != nil {
		log.Fatalf("Failed to list files, %v", err)
	}

	return deployedFilePermissions
}

func TestFeatureFilePermissionsOnLinux(t *testing.T) {
	packageFolderName := "feature-service-package"

	username := "root"
	password := "password"
	deployment := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: LinuxTestVirtualMachineUUID,
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "handler-test-service",
			Version: "*",
		},
		Account: &common.Account{Username: username, Password: password},
	}

	response, err := createFeatureDeploymentRequest(t, deployment, packageFolderName, false)
	if err != nil {
		t.Fatalf("Error creating Test Feature Deployment: %v", err)
	}
	defer func() {
		if err = createFeatureDeleteRequest(t, response.Identifier); err != nil {
			t.Fatalf("Error deleting Test Feature Deployment: %v", err)
		}
	}()

	auth := types.NamePasswordAuthentication{Username: username, Password: password}
	assets := getAssetsListFromPackage(t, packageFolderName)
	deployedFilePermissions := getDeployedAssetPermissions(t, LinuxTestVirtualMachineUUID, auth, assets)

	for index, asset := range assets {
		originalFilePermission := asset[2]
		deployedFilePermission := deployedFilePermissions[index]

		if len(originalFilePermission) == 4 && strings.HasPrefix(originalFilePermission, "0") {
			originalFilePermission = strings.TrimPrefix(originalFilePermission, "0")
		}

		if originalFilePermission != deployedFilePermission {
			log.Fatalf("File permissions do not match. Original: %v, Deployed: %v", originalFilePermission, deployedFilePermission)
		} else {
			log.Infof("File permissions match. Original: %v, Deployed: %v", originalFilePermission, deployedFilePermission)
		}
	}

	log.Infof("Feature stdout: %#v. Stderr: %#v", response.Stdout, response.Stderr)
}
