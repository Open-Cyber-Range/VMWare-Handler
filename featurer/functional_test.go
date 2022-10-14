package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
	"github.com/open-cyber-range/vmware-handler/library"
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

func uploadTestFeaturePackage() (err error) {
	uploadCommand := exec.Command("deputy", "publish")
	workingDirectory, err := os.Getwd()
	if err != nil {
		return
	}
	uploadCommand.Dir = path.Join(workingDirectory, "..", "extra", "test-deputy-packages", "test-feature-package")
	uploadCommand.Run()

	return
}

func TestFeatureDeployment(t *testing.T) {
	t.Parallel()
	configuration := startServer(3 * time.Second)

	ctx := context.Background()
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)

	// feature packages not fully implemented yet deputy side
	// err := uploadTestFeaturePackage()
	// if err != nil {
	// 	t.Fatalf("Failed to upload test feature package: %v", err)
	// }
	// log.Printf("Uploaded test package")

	feature := &feature.Feature{
		Name:             "test-feature",
		VirtualMachineId: "42127656-e390-d6a8-0703-c3425dbc8052", //debian - ok
		// VirtualMachineId: "4212d163-02a1-1aa2-12dc-aa822f9bd7dc", // opensuse - ok root Password123!@# vms
		// VirtualMachineId: "4212670f-6cfc-fccd-3c1b-598be72f5bf6", // dsl - not installed, unusable
		// VirtualMachineId: "42123aaa-89b4-bd8c-f677-dfd00282901d", // alpine 5 out of date
		// VirtualMachineId: "421268bf-097a-06ed-306f-a4282e0e09e5", // alpine 3 out of date root Password123!@# vms
		// VirtualMachineId: "4212de87-37d5-37b0-d1ad-398328e1c040", // alpine 2 out of date
		User:     "root",
		Password: "password",
		Source: &common.Source{
			Name:    "test-feature",
			Version: "*",
		},
	}
	result, err := gRPCClient.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to complete Feature Create request: %v", err)
	}

	log.Printf("Result: %v", result)

}
