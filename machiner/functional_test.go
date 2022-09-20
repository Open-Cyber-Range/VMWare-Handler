package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	virtual_machine "github.com/open-cyber-range/vmware-handler/grpc/virtual-machine"
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

var virtualMachineHardwareConfiguration = &virtual_machine.Configuration{
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

func exerciseCleanup(client *library.VMWareClient, exerciseFolder string, deploymentFolder string) (err error) {
	finder, _, _ := client.CreateFinderAndDatacenter()
	ctx := context.Background()

	virtualMachines, err := finder.VirtualMachineList(ctx, exerciseFolder+"/"+deploymentFolder+"/*")
	if err != nil && !strings.HasSuffix(err.Error(), "vm '"+exerciseFolder+"/"+deploymentFolder+"/*"+"' not found") {
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

	folder, err := finder.Folder(ctx, exerciseFolder)
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

func vmNodeExists(client *library.VMWareClient, exerciseName string, deploymentName string, nodeName string) bool {
	finder, _, _ := client.CreateFinderAndDatacenter()

	ctx := context.Background()
	virtualMachine, _ := finder.VirtualMachine(ctx, path.Join(testConfiguration.ExerciseRootPath, exerciseName, deploymentName, nodeName))
	return virtualMachine != nil
}

func creategRPCClient(t *testing.T, serverPath string) virtual_machine.VirtualMachineServiceClient {
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
	return virtual_machine.NewVirtualMachineServiceClient(connection)
}

func createExercise(t *testing.T, client *library.VMWareClient) (exerciseName string, deploymentName string, fullExercisePath string) {
	exerciseName = library.CreateRandomString(10)
	deploymentName = library.CreateRandomString(10)
	fullExercisePath = path.Join(testConfiguration.ExerciseRootPath, exerciseName)
	t.Cleanup(func() {
		cleanupError := exerciseCleanup(client, fullExercisePath, deploymentName)
		if cleanupError != nil {
			t.Fatalf("Failed to cleanup: %v", cleanupError)
		}
	})
	return
}

func createVmNode(t *testing.T, client virtual_machine.VirtualMachineServiceClient, exerciseName string, deploymentName string, vmwareClient *library.VMWareClient, links []string) *common.Identifier {
	templateVirtualMachine, err := vmwareClient.GetTemplateByName("debian10")
	if err != nil {
		t.Fatalf("Failed to find template by name: %v", err)
	}

	ctx := context.Background()
	virtualMachineDeployment := &virtual_machine.DeployVirtualMachine{
		VirtualMachine: &virtual_machine.VirtualMachine{
			Name:          "testNode",
			TemplateId:    templateVirtualMachine.UUID(ctx),
			Configuration: virtualMachineHardwareConfiguration,
			Links:         links,
		},
		MetaInfo: &common.MetaInfo{
			ExerciseName:   exerciseName,
			DeploymentName: deploymentName,
		},
	}

	result, err := client.Create(context.Background(), virtualMachineDeployment)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	if result.GetValue() == "" {
		t.Logf("Failed to retrieve UUID")
	}
	return result
}

func getVmConfigurations(client *library.VMWareClient, exerciseName string, deploymentName string, nodeName string) (managedVirtualMachine mo.VirtualMachine, err error) {
	finder, _, _ := client.CreateFinderAndDatacenter()

	ctx := context.Background()
	virtualMachine, _ := finder.VirtualMachine(ctx, path.Join("functional-tests", exerciseName, deploymentName, nodeName))

	err = virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &managedVirtualMachine)
	if err != nil {
		return managedVirtualMachine, err
	}
	return managedVirtualMachine, nil
}

func createAPIConfiguration() (apiConfiguration *swagger.Configuration) {
	apiConfiguration = swagger.NewConfiguration()
	apiConfiguration.BasePath = "https://" + switchTestConfiguration.NsxtApi + "/policy/api/v1"
	apiConfiguration.DefaultHeader["Authorization"] = fmt.Sprintf("Basic %v", switchTestConfiguration.NsxtAuth)
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
	policyTransportZoneListResult, _, err := segmentApiService.ListTransportZonesForEnforcementPoint(ctx, switchTestConfiguration.SiteId,
		"default", &swagger.ConnectivityApiListTransportZonesForEnforcementPointOpts{})
	if err != nil {
		return nil, err
	}
	for _, transportZone := range policyTransportZoneListResult.Results {
		if switchTestConfiguration.TransportZoneName == transportZone.DisplayName {
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

func createExerciseAndLinks(t *testing.T, client *library.VMWareClient, ctx context.Context, apiClient *swagger.APIClient, amountOfLinks int) (exerciseName string, deploymentName string, linkNames []string) {
	exerciseName = library.CreateRandomString(10)
	deploymentName = library.CreateRandomString(10)
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
		cleanupError := exerciseCleanup(client, fullExercisePath, deploymentName)
		if cleanupError != nil {
			t.Fatalf("Failed to cleanup: %v", cleanupError)
		}
		for _, linkName := range linkNames {
			segmentDeletionError := deleteLink(ctx, apiClient.SegmentsApi, linkName)
			if segmentDeletionError != nil {
				t.Fatalf("Could not delete segment: %v (%v)", linkName, segmentDeletionError)
			}
		}
	})
	return
}

func deleteLink(ctx context.Context, segmentApiService *swagger.SegmentsApiService, virtualSwitchName string) error {
	_, httpResponse, err := segmentApiService.ReadInfraSegment(ctx, virtualSwitchName)
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return fmt.Errorf("Segment not found for deleting, HTTP response: %v, error: %v", httpResponse, err)
	}
	_, err = segmentApiService.DeleteInfraSegment(ctx, virtualSwitchName)
	if err != nil {
		httpResponse, err := segmentApiService.ForceDeleteInfraSegment(ctx, virtualSwitchName, &swagger.SegmentsApiForceDeleteInfraSegmentOpts{})
		if err != nil {
			return fmt.Errorf("Segment not deleted, HTTP response: %v, error: %v", httpResponse, err)
		}
	}
	return nil
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
	exerciseName, deploymentName, _ := createExercise(t, &vmwareClient)
	createVmNode(t, gRPCClient, exerciseName, deploymentName, &vmwareClient, []string{})
	managedVirtualMachine, err := getVmConfigurations(&vmwareClient, exerciseName, deploymentName, "testNode")
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
	exerciseName, deploymentName, _ := createExercise(t, &vmwareClient)
	virtualMachineIdentifier := createVmNode(t, gRPCClient, exerciseName, deploymentName, &vmwareClient, []string{})

	gRPCClient.Delete(context.Background(), virtualMachineIdentifier)
	if vmNodeExists(&vmwareClient, exerciseName, deploymentName, "testNode") {
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
	exerciseName, deploymentName, _ := createExercise(t, &vmwareClient)

	identifier := createVmNode(t, gRPCClient, exerciseName, deploymentName, &vmwareClient, []string{})
	nodeExists := vmNodeExists(&vmwareClient, exerciseName, deploymentName, "testNode")

	if identifier.GetValue() == "" && nodeExists {
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
	if transportZoneError != nil {
		t.Fatalf("Couldn't get transport zone: %v", transportZoneError)
	}
	linkName, linkCreationError := createLink(ctx, transportZone, exerciseName, apiClient.SegmentsApi)
	if linkCreationError != nil {
		t.Fatalf("Couldn't create link: %v", linkCreationError)
	}
	linkDeletionError := deleteLink(ctx, apiClient.SegmentsApi, linkName)
	if linkCreationError != nil {
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
	exerciseName, deploymentName, linkNames := createExerciseAndLinks(t, &vmwareClient, ctx, apiClient, amountOfLinks)
	createVmNode(t, gRPCClient, exerciseName, deploymentName, &vmwareClient, linkNames)
	managedVirtualMachine, _ := getVmConfigurations(&vmwareClient, exerciseName, deploymentName, "testNode")

	if !vmNodeExists(&vmwareClient, exerciseName, deploymentName, "testNode") {
		t.Fatalf("Node does not exist")
	}

	checkError := vmwareClient.CheckVMLinks(ctx, managedVirtualMachine.Network, linkNames)
	if checkError != nil {
		t.Fatalf("Failed to check links: %v", checkError)
	}
}
