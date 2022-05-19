package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
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
	ServerAddress:      os.Getenv("TEST_VMWARE_SERVER_ADDRESS"),
}

func startServer(timeout time.Duration) (configuration Configuration) {
	configuration = testConfiguration
	rand.Seed(time.Now().UnixNano())
	randomPort := rand.Intn(10000) + 10000
	configuration.ServerAddress = fmt.Sprintf("127.0.0.1:%v", randomPort)
	go RealMain(&configuration)

	time.Sleep(timeout)
	return configuration
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
	finder, _, _ := createFinderAndDatacenter(client)
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

func vmNodeExists(client *govmomi.Client, exerciseName string, nodeName string) bool {
	finder, _, _ := createFinderAndDatacenter(client)

	ctx := context.Background()
	virtualMachine, _ := finder.VirtualMachine(ctx, testConfiguration.ExerciseRootPath+"/"+exerciseName+"/"+nodeName)
	return virtualMachine != nil
}

func creategRPCClient(t *testing.T, serverPath string) node.NodeServiceClient {
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

func createVmNode(t *testing.T, client node.NodeServiceClient, exerciseName string) *node.NodeIdentifier {
	nodeDeployment := &node.NodeDeployment{
		Parameters: &node.DeploymentParameters{
			Name:         "test-node",
			TemplateName: "debian10",
			ExerciseName: exerciseName,
		},
		Node: &node.Node{
			Identifier: &node.NodeIdentifier{
				NodeType: node.NodeType_vm,
			},
		},
	}

	resultNode, err := client.Create(context.Background(), nodeDeployment)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	if resultNode.GetIdentifier().GetValue() == "" {
		t.Logf("Failed to retrieve UUID")
	}
	return resultNode
}

func TestNodeDeletion(t *testing.T) {
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	VMWareClient, VMWareClientError := testConfiguration.createClient(ctx)
	if VMWareClientError != nil {
		t.Fatalf("Failed to send request: %v", VMWareClientError)
	}
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)
	exerciseName, _ := createExercise(t, VMWareClient)
	virtualMachineIdentifier := createVmNode(t, gRPCClient, exerciseName)

	gRPCClient.Delete(context.Background(), virtualMachineIdentifier)
	if vmNodeExists(VMWareClient, exerciseName, "test-node") {
		t.Fatalf("Node was not deleted")
	}
}

func TestNodeCreation(t *testing.T) {
	configuration := startServer(3 * time.Second)

	ctx := context.Background()
	VMWareClient, VMWareClientError := testConfiguration.createClient(ctx)
	if VMWareClientError != nil {
		t.Fatalf("Failed to send request: %v", VMWareClientError)
	}
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)
	exerciseName, _ := createExercise(t, VMWareClient)

	nodeIdentifier := createVmNode(t, gRPCClient, exerciseName)
	nodeExists := vmNodeExists(VMWareClient, exerciseName, "test-node")

	if nodeIdentifier.GetIdentifier().GetValue() == "" && nodeExists {
		t.Fatalf("Node exists but failed to retrieve UUID")
	}
	if !nodeExists {
		t.Fatalf("Node was not created")
	}
}

func createNodeDeploymentOfTypeSwitch() *node.NodeDeployment {
	nodeDeployment := &node.NodeDeployment{
		Parameters: &node.DeploymentParameters{
			Name:         fmt.Sprintf("test-virtual-switch-%v", createRandomString(5)),
			ExerciseName: createRandomString(10),
		},
		Node: &node.Node{
			Identifier: &node.NodeIdentifier{
				NodeType: node.NodeType_switch,
			},
		},
	}
	return nodeDeployment
}

func virtualSwitchExists(t *testing.T, ctx context.Context, nsxtClient *nsxt.APIClient, virtualSwitchUuid string) bool {
	_, httpResponse, err := nsxtClient.LogicalSwitchingApi.GetLogicalSwitch(ctx, virtualSwitchUuid)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	return httpResponse.StatusCode == http.StatusOK
}

func TestVirtualSwitchCreation(t *testing.T) {
	ctx := context.Background()
	configuration := nsxt.NewConfiguration()
	configuration.Host = os.Getenv("NSXT_API")
	configuration.DefaultHeader["Authorization"] = os.Getenv("NSXT_AUTH")
	configuration.Insecure = true
	nsxtClient, err := nsxt.NewAPIClient(configuration)
	if err != nil {
		t.Fatalf("Failed create new API client: %v", err)
	}
	testVirtualSwitch, err := CreateVirtualSwitch(ctx, createNodeDeploymentOfTypeSwitch())
	if err != nil {
		t.Fatalf("Failed create new virtual switch: %v", err)
	}

	if !virtualSwitchExists(t, ctx, nsxtClient, testVirtualSwitch.GetIdentifier().GetValue()) {
		t.Fatalf("Virtual switch was not created")
	} else {
		_, err := nsxtClient.LogicalSwitchingApi.DeleteLogicalSwitch(ctx, testVirtualSwitch.GetIdentifier().GetValue(), nil)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		if virtualSwitchExists(t, ctx, nsxtClient, testVirtualSwitch.GetIdentifier().GetValue()) {
			t.Logf("Test Virtual switch \" %v \" was not cleaned up", testVirtualSwitch.GetIdentifier().GetValue())
		}
	}
}
