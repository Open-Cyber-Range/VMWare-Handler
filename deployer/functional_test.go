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

func exerciseCleanup(client *govmomi.Client, folderPath string) (err error) {
	finder := CreateFinder(client)
	ctx := context.Background()

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

func nodeExists(client *govmomi.Client, exerciseName string, nodeName string) bool {
	finder := CreateFinder(client)

	ctx := context.Background()
	virtualMachine, _ := finder.VirtualMachine(ctx, testConfiguration.ExerciseRootPath+"/"+exerciseName+"/"+nodeName)
	return virtualMachine != nil
}

func creategRPCClient(t *testing.T) node.NodeServiceClient {
	connection, connectionError := grpc.Dial(testConfiguration.ServerPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		t.Fatalf("did not connect: %v", connectionError)
	}
	t.Cleanup(func() {
		connectionError := connection.Close()
		if connectionError != nil {
			t.Fatalf("Failed to close connection: %v", connectionError)
		}
	})
	return node.NewNodeServiceClient(connection)
}

func createExercise(t *testing.T, client *govmomi.Client) (exerciseName string, fullExercisePath string) {
	exerciseName = createRandomString(10)
	fullExercisePath = testConfiguration.ExerciseRootPath + "/" + exerciseName
	t.Cleanup(func() {
		cleanupError := exerciseCleanup(client, fullExercisePath)
		if cleanupError != nil {
			t.Fatalf("Failed to cleanup: %v", cleanupError)
		}
	})
	return
}

func TestNodeDeployment(t *testing.T) {
	startServer(1 * time.Second)

	ctx := context.Background()
	VMWareClient, VMWareClientError := testConfiguration.createClient(ctx)
	if VMWareClientError != nil {
		t.Fatalf("Failed to send request: %v", VMWareClientError)
	}
	gRPCClient := creategRPCClient(t)
	exerciseName, _ := createExercise(t, VMWareClient)

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
	if !nodeExists(VMWareClient, exerciseName, "test-node") {
		t.Fatalf("Node was not created")
	}
}
