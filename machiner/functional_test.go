package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
	"github.com/vmware/govmomi/vim25/mo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
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

var virtualMachineHardwareConfiguration = &node.Configuration{
	Cpu: 2,
	Ram: 2147483648, // 2048mb
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

func exerciseCleanup(client *library.VMWareClient, folderPath string) (err error) {
	finder, _, _ := client.CreateFinderAndDatacenter()
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

func vmNodeExists(client *library.VMWareClient, exerciseName string, nodeName string) bool {
	finder, _, _ := client.CreateFinderAndDatacenter()

	ctx := context.Background()
	virtualMachine, _ := finder.VirtualMachine(ctx, path.Join(testConfiguration.ExerciseRootPath, exerciseName, nodeName))
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

func createExercise(t *testing.T, client *library.VMWareClient) (exerciseName string, fullExercisePath string) {
	exerciseName = library.CreateRandomString(10)
	fullExercisePath = path.Join(testConfiguration.ExerciseRootPath, exerciseName)
	t.Cleanup(func() {
		cleanupError := exerciseCleanup(client, fullExercisePath)
		if cleanupError != nil {
			t.Fatalf("Failed to cleanup: %v", cleanupError)
		}
	})
	return
}

func createVmNode(t *testing.T, client node.NodeServiceClient, exerciseName string, vmwareClient *library.VMWareClient) *node.NodeIdentifier {
	templateVirtualMachine, err := vmwareClient.GetTemplateByName("debian10")
	if err != nil {
		t.Fatalf("Failed to find template by name: %v", err)
	}

	ctx := context.Background()
	nodeDeployment := &node.NodeDeployment{
		Parameters: &node.DeploymentParameters{
			Name:         "test-node",
			TemplateId:   templateVirtualMachine.UUID(ctx),
			ExerciseName: exerciseName,
			Links:        []string{"TEST1", "TEST2"},
		},
		Node: &node.Node{
			Identifier: &node.NodeIdentifier{
				NodeType: node.NodeType_vm,
			},
			Configuration: virtualMachineHardwareConfiguration,
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

func getVmConfigurations(client *library.VMWareClient, exerciseName string, nodeName string) (managedVirtualMachine mo.VirtualMachine, err error) {
	finder, _, _ := client.CreateFinderAndDatacenter()

	ctx := context.Background()
	virtualMachine, _ := finder.VirtualMachine(ctx, path.Join("functional-tests", exerciseName, nodeName))

	err = virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &managedVirtualMachine)
	if err != nil {
		return managedVirtualMachine, err
	}
	return managedVirtualMachine, nil
}

func TestVerifyNodeCpuAndMemory(t *testing.T) {
	t.Parallel()
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	govmomiClient, govmomiClientError := testConfiguration.CreateClient(ctx)
	if govmomiClientError != nil {
		t.Fatalf("Failed to send request: %v", govmomiClientError)
	}
	vmwareClient := library.NewVMWareClient(govmomiClient, testConfiguration.TemplateFolderPath)
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)
	exerciseName, _ := createExercise(t, &vmwareClient)
	createVmNode(t, gRPCClient, exerciseName, &vmwareClient)
	managedVirtualMachine, err := getVmConfigurations(&vmwareClient, exerciseName, "test-node")
	if err != nil {
		t.Fatalf("Failed to retrieve VM configuration: %v", err)
	}
	if managedVirtualMachine.Config.Hardware.NumCPU != int32(virtualMachineHardwareConfiguration.Cpu) {
		t.Fatalf("Expected %v CPUs, got %v", virtualMachineHardwareConfiguration.Cpu, managedVirtualMachine.Config.Hardware.NumCPU)
	}
	if managedVirtualMachine.Config.Hardware.MemoryMB != int32(virtualMachineHardwareConfiguration.Ram>>20) {
		t.Fatalf("Expected %v of memory, got %v", (virtualMachineHardwareConfiguration.Ram >> 20), managedVirtualMachine.Config.Hardware.MemoryMB)
	}
}

func TestNodeDeletion(t *testing.T) {
	t.Parallel()
	configuration := startServer(3 * time.Second)
	ctx := context.Background()
	govmomiClient, govmomiClientError := testConfiguration.CreateClient(ctx)
	if govmomiClientError != nil {
		t.Fatalf("Failed to create govmomi client: %v", govmomiClientError)
	}
	vmwareClient := library.NewVMWareClient(govmomiClient, testConfiguration.TemplateFolderPath)
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)
	exerciseName, _ := createExercise(t, &vmwareClient)
	virtualMachineIdentifier := createVmNode(t, gRPCClient, exerciseName, &vmwareClient)

	gRPCClient.Delete(context.Background(), virtualMachineIdentifier)
	if vmNodeExists(&vmwareClient, exerciseName, "test-node") {
		t.Fatalf("Node was not deleted")
	}
}

func TestNodeCreation(t *testing.T) {
	t.Parallel()
	configuration := startServer(3 * time.Second)

	ctx := context.Background()
	govmomiClient, govmomiClientError := testConfiguration.CreateClient(ctx)
	if govmomiClientError != nil {
		t.Fatalf("Failed to send request: %v", govmomiClientError)
	}
	vmwareClient := library.NewVMWareClient(govmomiClient, testConfiguration.TemplateFolderPath)
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)
	exerciseName, _ := createExercise(t, &vmwareClient)

	nodeIdentifier := createVmNode(t, gRPCClient, exerciseName, &vmwareClient)
	nodeExists := vmNodeExists(&vmwareClient, exerciseName, "test-node")

	if nodeIdentifier.GetIdentifier().GetValue() == "" && nodeExists {
		t.Fatalf("Node exists but failed to retrieve UUID")
	}
	if !nodeExists {
		t.Fatalf("Node was not created")
	}
}

func TestSwitcherCapability(t *testing.T) {
	t.Parallel()
	serverConfiguration := startServer(time.Second * 3)
	ctx := context.Background()
	capabilityClient := library.CreateCapabilityClient(t, serverConfiguration.ServerAddress)
	handlerCapabilities, err := capabilityClient.GetCapabilities(ctx, new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Failed to get deployer capability: %v", err)
	}
	handlerCapability := handlerCapabilities.GetValues()[0]
	if handlerCapability.Number() != capability.Capabilities_VirtualMachine.Number() {
		t.Fatalf("Capability service returned incorrect value: expected: %v, got: %v", capability.Capabilities_VirtualMachine.Enum(), handlerCapability)
	}
}
