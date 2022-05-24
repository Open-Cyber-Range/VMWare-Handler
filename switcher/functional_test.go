package switcher

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	deployer "github.com/open-cyber-range/vmware-node-deployer/deployer"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var testConfiguration = deployer.Configuration{
	NsxtApi:           os.Getenv("TEST_NSXT_API"),
	NsxtAuth:          fmt.Sprintf("Basic %v", os.Getenv("TEST_NSXT_AUTH")),
	TransportZoneName: os.Getenv("TEST_NSXT_TRANSPORT_ZONE_NAME"),
	ServerAddress:     os.Getenv("TEST_VMWARE_SERVER_ADDRESS"),
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
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("Failed to send request: %v", err)
	}
	return httpResponse.StatusCode == http.StatusOK
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
func startServer(timeout time.Duration) (configuration deployer.Configuration) {
	configuration = testConfiguration
	rand.Seed(time.Now().UnixNano())
	randomPort := rand.Intn(10000) + 10000
	configuration.ServerAddress = fmt.Sprintf("127.0.0.1:%v", randomPort)
	go RealMain(&configuration)

	time.Sleep(timeout)
	return configuration
}

func TestVirtualSwitchCreation(t *testing.T) {
	serverConfiguration := startServer(time.Second * 3)
	ctx := context.Background()

	nsxtConfiguration := nsxt.NewConfiguration()
	nsxtConfiguration.Host = serverConfiguration.NsxtApi
	nsxtConfiguration.DefaultHeader["Authorization"] = serverConfiguration.NsxtAuth
	nsxtConfiguration.Insecure = true
	nsxtClient, err := nsxt.NewAPIClient(nsxtConfiguration)
	if err != nil {
		t.Fatalf("Failed create new API client: %v", err)
	}

	gRPCClient := creategRPCClient(t, serverConfiguration.ServerAddress)
	server := grpc.NewServer()
	node.RegisterNodeServiceServer(server, &nsxtNodeServer{
		Client:        nsxtClient,
		Configuration: &serverConfiguration,
	})

	nodeDeployment := createNodeDeploymentOfTypeSwitch()

	testVirtualSwitch, err := gRPCClient.Create(ctx, nodeDeployment)
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
