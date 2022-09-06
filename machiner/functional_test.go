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
	"net/http"
	"crypto/tls"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
	swagger "github.com/open-cyber-range/vmware-handler/nsx_t_openapi"
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
	NsxtApi:            os.Getenv("TEST_NSXT_API"),
	NsxtAuth:           os.Getenv("TEST_NSXT_AUTH"),
	TransportZoneName:  os.Getenv("TEST_NSXT_TRANSPORT_ZONE_NAME"),
	SiteId:				"default",
}

var virtualMachineHardwareConfiguration = &node.Configuration{
	Cpu: 2,
	Ram: 1073741824, // 1024mb
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

func createVmNode(t *testing.T, client node.NodeServiceClient, exerciseName string, vmwareClient *library.VMWareClient, links []string) *node.NodeIdentifier {
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
			Links:        links,
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

func createAPIConfiguration() (apiConfiguration *swagger.Configuration) {
	apiConfiguration = swagger.NewConfiguration()
	apiConfiguration.BasePath = "https://" + testConfiguration.NsxtApi + "/policy/api/v1"
	apiConfiguration.DefaultHeader["Authorization"] = fmt.Sprintf("Basic %v", testConfiguration.NsxtAuth)
	apiConfiguration.HTTPClient = &http.Client{
		// TODO TLS still insecure
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: testConfiguration.Insecure},
		},
	}
	return
}

func createAPIClient() (apiClient *swagger.APIClient) {
	apiConfiguration := createAPIConfiguration()
	apiClient = swagger.NewAPIClient(apiConfiguration)
	return
}

func getTransportZone(ctx context.Context, apiClient *swagger.APIClient) (transportZone *swagger.PolicyTransportZone, err error) {
	segmentApiService := apiClient.ConnectivityApi
	policyTransportZoneListResult, _, err := segmentApiService.ListTransportZonesForEnforcementPoint(ctx, testConfiguration.SiteId,
		"default", &swagger.ConnectivityApiListTransportZonesForEnforcementPointOpts{})
	if err != nil {
		return nil, err
	}
	for _, transportZone := range policyTransportZoneListResult.Results {
		if testConfiguration.TransportZoneName == transportZone.DisplayName {
			return &transportZone, nil
		}
	}
	return nil, fmt.Errorf("Transport zone not in list")
}

func createLink(ctx context.Context, transportZone *swagger.PolicyTransportZone, exerciseName string, segmentApiService *swagger.SegmentsApiService) (linkName string, err error) {
	var segment = swagger.Segment{
		Id: fmt.Sprintf("test-virtual-switch-%v", library.CreateRandomString(5)) + "_" +
			exerciseName + "_" + transportZone.Id,
		TransportZonePath: transportZone.Path,
	}
	segmentResponse, httpResponse, err := segmentApiService.CreateOrReplaceInfraSegment(ctx, segment.Id, segment)
	if err != nil {
		return "", fmt.Errorf("API request to create segment failed (%v)", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Segment not created (%v)", httpResponse.Status)
	}
	return segmentResponse.DisplayName, nil
}

func createExerciseAndLinks(t *testing.T, client *library.VMWareClient, ctx context.Context, apiClient *swagger.APIClient, amountOfLinks int) (exerciseName string, linkNames []string) {
	exerciseName = library.CreateRandomString(10)
	fullExercisePath := path.Join(testConfiguration.ExerciseRootPath, exerciseName)

	transportZone, err := getTransportZone(ctx, apiClient)
	if err != nil {
		t.Fatalf("Could not find transport zone: %v", err)
	}
	for i := 0; i < amountOfLinks; i++ {
		linkName, err := createLink(ctx, transportZone, exerciseName, apiClient.SegmentsApi)
		if err != nil {
			t.Fatalf("Link creation failed: %v", err)
		}
		linkNames = append(linkNames, linkName)
	}
	t.Cleanup(func() {
		cleanupError := exerciseCleanup(client, fullExercisePath)
		if cleanupError != nil {
			t.Fatalf("Failed to cleanup: %v", cleanupError)
		}
		for _, linkName := range linkNames {
			segmentDeletionError := deleteLink(ctx, apiClient, linkName)
			if segmentDeletionError != nil {
				t.Fatalf("Could not delete segment: %v (%v)", linkName, segmentDeletionError)
			}
		}
	})
	return
}

func deleteLink(ctx context.Context, apiClient *swagger.APIClient, virtualSwitchName string) error {
	segmentApiService := apiClient.SegmentsApi
	_, err := segmentApiService.DeleteInfraSegment(ctx, virtualSwitchName)
	if err != nil {
		httpResponse, err := segmentApiService.ForceDeleteInfraSegment(ctx, virtualSwitchName, &swagger.SegmentsApiForceDeleteInfraSegmentOpts{})	
		if err != nil {
			return fmt.Errorf("Segment not deleted, HTTP response: %v, error: %v", httpResponse, err)
		}
	}
	return nil
}

func checkVMLinks(client *library.VMWareClient, ctx context.Context, managedVirtualMachine mo.VirtualMachine, linkNames []string) error {
	vmNetworks := managedVirtualMachine.Network
	if vmNetworks == nil {
		return fmt.Errorf("Failed to retrieve VM network list")
	}

	var vmNetworkNames string
	for _, network := range vmNetworks {
		vmNetworkNames = vmNetworkNames + " " + network.Value
	}

	networkNames, err := findLinks(client, ctx, linkNames)
	if err != nil {
		return err
	}

	for _, networkName := range networkNames {
		if !strings.Contains(vmNetworkNames, networkName) {
			return fmt.Errorf("Link %v is not added to VM", networkName)
		}
	}
	return nil
}

func findLinks(client *library.VMWareClient, ctx context.Context, linkNames []string) (networkNames []string, err error) {
	finder, _, _ := client.CreateFinderAndDatacenter()
	for _, linkName := range linkNames {
		network, err := finder.Network(ctx, linkName)
		if err != nil {
			return nil, fmt.Errorf("Failed to find network %v (%v)", linkName, err)
		}
		networkNames = append(networkNames, network.Reference().Value)
	}
	return networkNames, nil
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
	createVmNode(t, gRPCClient, exerciseName, &vmwareClient, []string{})
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
	virtualMachineIdentifier := createVmNode(t, gRPCClient, exerciseName, &vmwareClient, []string{})

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

	nodeIdentifier := createVmNode(t, gRPCClient, exerciseName, &vmwareClient, []string{})
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

func TestLinkCreationAndDeletion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	exerciseName := library.CreateRandomString(10)
	apiClient := createAPIClient()
	transportZone, transportZoneError := getTransportZone(ctx, apiClient)
	if transportZoneError!= nil {
		t.Fatalf("Couldn't get transport zone: %v", transportZoneError)
	}
	linkName, linkCreationError := createLink(ctx, transportZone, exerciseName, apiClient.SegmentsApi)
	if linkCreationError!= nil {
		t.Fatalf("Couldn't create link: %v", linkCreationError)
	}
	linkDeletionError := deleteLink(ctx, apiClient, linkName)
	if linkCreationError!= nil {
		t.Fatalf("Couldn't delete link: %v", linkDeletionError)
	}
}

func TestLinkAddition(t *testing.T) {
	t.Parallel()
	configuration := startServer(3 * time.Second)

	ctx := context.Background()
	govmomiClient, govmomiClientError := testConfiguration.CreateClient(ctx)
	if govmomiClientError != nil {
		t.Fatalf("Failed to send request: %v", govmomiClientError)
	}
	vmwareClient := library.NewVMWareClient(govmomiClient, testConfiguration.TemplateFolderPath)
	gRPCClient := creategRPCClient(t, configuration.ServerAddress)
	apiClient := createAPIClient()
	amountOfLinks := 3
	exerciseName, linkNames := createExerciseAndLinks(t, &vmwareClient, ctx, apiClient, amountOfLinks)
	createVmNode(t, gRPCClient, exerciseName, &vmwareClient, linkNames)
	managedVirtualMachine, _ := getVmConfigurations(&vmwareClient, exerciseName, "test-node")

	if !vmNodeExists(&vmwareClient, exerciseName, "test-node") {
		t.Fatalf("Node does not exist")
	}

	checkError := checkVMLinks(&vmwareClient, ctx, managedVirtualMachine, linkNames)
	if checkError != nil {
		t.Fatalf("Failed to check links: %v", checkError)
	}
}
