package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
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

type TestNSXTConfiguration struct {
	NsxtApi           string
	NsxtAuth          string
	TransportZoneName string
	SiteId            string
}

var switchTestConfiguration = TestNSXTConfiguration{
	NsxtApi:           os.Getenv("TEST_NSXT_API"),
	NsxtAuth:          os.Getenv("TEST_NSXT_AUTH"),
	TransportZoneName: os.Getenv("TEST_NSXT_TRANSPORT_ZONE_NAME"),
	SiteId:            "default",
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

func creategRPCClient(t *testing.T, serverPath string) feature.FeatureServiceClient {
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

func createRedisClient() library.Storage {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     testConfiguration.RedisAddress,
		Password: testConfiguration.RedisPassword,
		DB:       0,
	})

	return library.NewStorage(redisClient)
}

func createFeatureDeploymentRequest(t *testing.T, feature *feature.Feature, packageName string, accounts []library.Account) {
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)
	storage := createRedisClient()

	if err := library.PublishTestPackage(packageName); err != nil {
		t.Fatalf("Failed to upload test feature package: %v", err)
	}

	if err := library.Create(ctx, storage.RedisClient, feature.TemplateId, accounts); err != nil {
		t.Fatalf("Test Create redis entry error: %v", err)
	}

	response, err := gRPCClient.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Test Create request error: %v", err)
	}

	log.Infof("Feature Create finished, id: %v", response.Identifier.GetValue())
	_, err = gRPCClient.Delete(ctx, response.Identifier)
	if err != nil {
		t.Fatalf("Test Delete request error: %v", err)
	}
	log.Infof("Feature delete finished")
}

func TestFeatureDeploymentAndDeletionOnLinux(t *testing.T) {
	t.Parallel()

	packageName := "feature-service-package"
	accounts := []library.Account{{Name: "root", Password: "password"}}

	feature := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: "42127656-e390-d6a8-0703-c3425dbc8052",
		FeatureType:      feature.FeatureType_service,
		Username:         "root",
		Source: &common.Source{
			Name:    "test-service",
			Version: "0.3.0",
		},
		TemplateId: "test-template-id-1",
	}
	createFeatureDeploymentRequest(t, feature, packageName, accounts)
}

func TestFeatureDeploymentAndDeletionOnWindows(t *testing.T) {
	t.Skip("Skipped until release of Deputy that contains 'Accounts' and 'Action' field")
	t.Parallel()

	packageName := "feature-win-service-package"
	accounts := []library.Account{{Name: "user", Password: "password"}}

	feature := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: "42122b12-3a17-c0fb-eb3c-7cd935bb595b",
		Username:         "user",
		FeatureType:      feature.FeatureType_service,
		Source: &common.Source{
			Name:    "test-windows-service",
			Version: "*",
		},
		TemplateId: "test-template-id-2",
	}
	createFeatureDeploymentRequest(t, feature, packageName, accounts)
}
