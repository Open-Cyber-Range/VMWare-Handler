package main

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var testConfiguration = Configuration{
	User:               os.Getenv("TEST_VMWARE_USER"),
	Password:           os.Getenv("TEST_VMWARE_PASSWORD"),
	Hostname:           os.Getenv("TEST_VMWARE_HOSTNAME"),
	Insecure:           true,
	TemplateFolderPath: os.Getenv("TEST_VMWARE_TEMPLATE_FOLDER_PATH"),
	ResourcePoolPath:   os.Getenv("TEST_VMWARE_RESOURCE_POOL_PATH"),
	ExerciseRootPath:   os.Getenv("TEST_VMWARE_EXERCISE_ROOT_PATH"),
	ServerPath:         os.Getenv("TEST_VMWARE_SERVER_PATH"),
}

func startServer(timeout time.Duration) {
	go RealMain(&testConfiguration)

	time.Sleep(timeout)
}

func createRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func testCleanup(client *govmomi.Client, folderPath string) (err error) {
	finder := find.NewFinder(client.Client, true)
	ctx := context.Background()
	datacenter, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return
	}
	finder.SetDatacenter(datacenter)

	virtualMachines, err := finder.VirtualMachineList(ctx, folderPath+"/*")
	if err != nil && !strings.HasSuffix(err.Error(), "vm '"+folderPath+"/*"+"' not found") {
		return
	}

	for _, virtualMachine := range virtualMachines {
		powerOffTask, powerOffError := virtualMachine.PowerOff(ctx)
		if powerOffError != nil {
			return powerOffError
		}
		powerOffTask.Wait(ctx)
		destroyMachineTask, destroyError := virtualMachine.Destroy(ctx)
		if destroyError != nil {
			return destroyError
		}
		destroyMachineTask.Wait(ctx)
	}

	folder, err := finder.Folder(ctx, folderPath)
	if err != nil {
		return
	}
	destroyFolderTask, err := folder.Destroy(ctx)
	if err != nil {
		return
	}
	destroyFolderTask.Wait(ctx)
	return nil
}

func TestNodeDeployment(t *testing.T) {
	startServer(1 * time.Second)

	ctx := context.Background()
	VMWareClient, VMWareClientError := testConfiguration.createClient(ctx)
	if VMWareClientError != nil {
		t.Fatalf("Failed to send request: %v", VMWareClientError)
	}
	connection, err := grpc.Dial(testConfiguration.ServerPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer func() {
		connectionError := connection.Close()
		if connectionError != nil {
			t.Fatalf("Failed to close connection: %v", connectionError)
		}
	}()
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	gRPCClient := node.NewNodeServiceClient(connection)
	exerciseName := createRandomString(10)
	defer func() {
		cleanupError := testCleanup(VMWareClient, testConfiguration.ExerciseRootPath+"/"+exerciseName)
		if cleanupError != nil {
			t.Fatalf("Failed to cleanup: %v", cleanupError)
		}
	}()
	reply, err := gRPCClient.Create(context.Background(), &node.Node{
		Name:         "test-node",
		TemplateName: "debian10",
		ExerciseName: exerciseName,
	})
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	if reply.Status != common.SimpleResponse_OK {
		t.Fatalf("Failed to create node: %v", reply.Message)
	}
}
