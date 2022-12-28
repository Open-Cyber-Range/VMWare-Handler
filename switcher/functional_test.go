package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	swithc_grpc "github.com/open-cyber-range/vmware-handler/grpc/switch"
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

func createNodeDeploymentOfTypeSwitch() *swithc_grpc.DeploySwitch {
	switchDeployment := &swithc_grpc.DeploySwitch{
		MetaInfo: &common.MetaInfo{
			DeploymentName: library.CreateRandomString(10),
			ExerciseName:   library.CreateRandomString(10),
		},
		Switch: &swithc_grpc.Switch{
			Name: fmt.Sprintf("test-virtual-switch-%v", library.CreateRandomString(5)),
		},
	}
	return switchDeployment
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
	t.Skip()
	ctx := context.Background()
	switchDeployment := createNodeDeploymentOfTypeSwitch()
	serverConfiguration := startServer(time.Second * 3)
	apiClient := createAPIClient(&serverConfiguration)
	var nsxtNodeServer = switchServer{
		UnimplementedSwitchServiceServer: swithc_grpc.UnimplementedSwitchServiceServer{},
		APIClient:                        apiClient,
		Configuration:                    serverConfiguration,
	}
	var deploySwitch = DeploySwitch{
		DeploymentMessge: switchDeployment,
		APIClient:        apiClient,
		Configuration:    serverConfiguration,
	}
	segment, err := deploySwitch.createNetworkSegment(ctx)
	if err != nil {
		t.Fatalf("Failed to create network segment: %v", err)
	}
	err = nsxtNodeServer.deleteAndVerifyInfraSegment(ctx, segment.Id)
	if err != nil {
		t.Fatalf("Failed to delete network segment: %v", err)
	}
}
