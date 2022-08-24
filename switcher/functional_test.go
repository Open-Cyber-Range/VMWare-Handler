package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var testConfiguration = Configuration{
	NsxtApi:           os.Getenv("TEST_NSXT_API"),
	NsxtAuth:          os.Getenv("TEST_NSXT_AUTH"),
	TransportZoneName: os.Getenv("TEST_NSXT_TRANSPORT_ZONE_NAME"),
	ServerAddress:     "127.0.0.1",
	Insecure:          true,
}

func createNodeDeploymentOfTypeSwitch() *node.NodeDeployment {
	nodeDeployment := &node.NodeDeployment{
		Parameters: &node.DeploymentParameters{
			Name:         fmt.Sprintf("test-virtual-switch-%v", library.CreateRandomString(5)),
			ExerciseName: library.CreateRandomString(10),
		},
		Node: &node.Node{
			Identifier: &node.NodeIdentifier{
				NodeType: node.NodeType_switch,
			},
		},
	}
	return nodeDeployment
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

func createCapabilityClient(t *testing.T, serverPath string) capability.CapabilityClient {
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
	return capability.NewCapabilityClient(connection)
}

func startServer(timeout time.Duration) (configuration Configuration) {
	configuration = testConfiguration
	rand.Seed(time.Now().UnixNano())
	randomPort := rand.Intn(10000) + 10000
	configuration.ServerAddress = fmt.Sprintf("%v:%v", configuration.ServerAddress, randomPort)
	go RealMain(&configuration)

	time.Sleep(timeout)
	return configuration
}

func createVirtualSwitch(t *testing.T, serverConfiguration Configuration) (*node.NodeIdentifier, error) {
	gRPCClient := creategRPCClient(t, serverConfiguration.ServerAddress)
	nodeDeployment := createNodeDeploymentOfTypeSwitch()
	testVirtualSwitch, err := gRPCClient.Create(context.Background(), nodeDeployment)
	if err != nil {
		return nil, err
	}
	return testVirtualSwitch, nil
}

func TestVirtualSwitchCreationAndDeletion(t *testing.T) {
	serverConfiguration := startServer(time.Second * 3)
	ctx := context.Background()
	gRPCClient := creategRPCClient(t, serverConfiguration.ServerAddress)
	nodeIdentifier, err := createVirtualSwitch(t, serverConfiguration)
	if err != nil {
		t.Fatalf("Failed to create new virtual switch: %v", err)
	}
	_, err = gRPCClient.Delete(ctx, nodeIdentifier)
	if err != nil {
		t.Fatalf("Failed to delete test virtual switch: %v", err)
	}
}

func TestSwitcherCapability(t *testing.T) {
	t.Parallel()
	serverConfiguration := startServer(time.Second * 3)
	ctx := context.Background()
	capabilityClient := createCapabilityClient(t, serverConfiguration.ServerAddress)
	handlerCapabilities, err := capabilityClient.GetCapabilities(ctx, new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Failed to get deployer capability: %v", err)
	}
	handlerCapability := handlerCapabilities.GetValues()[0]
	if handlerCapability.Number() != capability.Capabilities_Switch.Number() {
		t.Fatalf("Capability service returned incorrect value: expected: %v, got: %v", capability.Capabilities_Switch.Enum(), handlerCapability.Enum())
	}
}

func TestSegmentCreationAndDeletion(t *testing.T) {
	ctx := context.Background()
	nodeDeployment := createNodeDeploymentOfTypeSwitch()
	serverConfiguration := startServer(time.Second * 3)
	apiClient := createAPIClient(&serverConfiguration)
	var nsxtNodeServer = nsxtNodeServer{
		UnimplementedNodeServiceServer: node.UnimplementedNodeServiceServer{},
		APIClient:                      apiClient,
		Configuration:                  serverConfiguration,
	}
	segment, err := createNetworkSegment(ctx, nodeDeployment, &nsxtNodeServer)
	if err != nil {
		t.Fatalf("Failed to create network segment: %v", err)
	}
	err = deleteAndVerifyInfraSegment(ctx, segment.Id, &nsxtNodeServer)
	if err != nil {
		t.Fatalf("Failed to delete network segment: %v", err)
	}
}
