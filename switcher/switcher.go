package switcher

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	nsxtCommon "github.com/ScottHolden/go-vmware-nsxt/common"
	"github.com/ScottHolden/go-vmware-nsxt/manager"
	deployer "github.com/open-cyber-range/vmware-node-deployer/deployer"
	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func createNsxtClient(serverConfiguration *deployer.Configuration) (nsxtClient *nsxt.APIClient, err error) {
	nsxtConfiguration := nsxt.NewConfiguration()
	if serverConfiguration.NsxtApi != "" && serverConfiguration.NsxtAuth != "" && serverConfiguration.TransportZoneName != "" {
		nsxtConfiguration.Host = serverConfiguration.NsxtApi
		nsxtConfiguration.DefaultHeader["Authorization"] = serverConfiguration.NsxtAuth
	} else {
		return nil, status.Error(codes.InvalidArgument, "NSX-T API, Authorization key and Transport Zone Name must be set")
	}
	nsxtConfiguration.Insecure = true
	nsxtClient, err = nsxt.NewAPIClient(nsxtConfiguration)
	if err != nil {
		return
	}
	return
}

func findTransportZoneIdByName(ctx context.Context, nsxtClient *nsxt.APIClient, serverConfiguration *deployer.Configuration) (string, error) {
	transportZones, _, err := nsxtClient.NetworkTransportApi.ListTransportZones(ctx, nil)
	if err != nil {
		status.New(codes.Internal, fmt.Sprintf("CreateVirtualSwitch: ListTransportZones error (%v)", err))
		return "", err
	}
	for i, transportNode := range transportZones.Results {
		if strings.EqualFold(transportZones.Results[i].DisplayName, serverConfiguration.TransportZoneName) {
			return transportNode.Id, nil
		}
	}
	return "", status.Error(codes.InvalidArgument, "Transport zone not found")
}

type nsxtNodeServer struct {
	node.UnimplementedNodeServiceServer
	Client        *nsxt.APIClient
	Configuration *deployer.Configuration
}

func (server *nsxtNodeServer) Create(ctx context.Context, nodeDeployment *node.NodeDeployment) (identifier *node.NodeIdentifier, err error) {
	transportZoneId, err := findTransportZoneIdByName(ctx, server.Client, server.Configuration)
	if err != nil {
		return
	}
	newVirtualSwitch := manager.LogicalSwitch{
		TransportZoneId: transportZoneId,
		DisplayName:     nodeDeployment.GetParameters().GetName(),
		AdminState:      "UP",
		ReplicationMode: "MTEP",
		Description:     fmt.Sprintf("Created for exercise: %v", nodeDeployment.GetParameters().GetExerciseName()),
		Tags:            []nsxtCommon.Tag{{Scope: "policyPath", Tag: fmt.Sprintf("/infra/segments/%v", nodeDeployment.GetParameters().GetName())}},
	}

	virtualSwitch, httpResponse, err := server.Client.LogicalSwitchingApi.CreateLogicalSwitch(ctx, newVirtualSwitch)
	if err != nil {
		status.New(codes.Internal, fmt.Sprintf("CreateVirtualSwitch: API request error (%v)", err))
		return
	}
	if httpResponse.StatusCode != http.StatusCreated {
		status.New(codes.Internal, fmt.Sprintf("CreateVirtualSwitch: Virtual Switch not created (%v)", httpResponse.Status))
		return
	}

	status.New(codes.OK, "Virtual Switch creation successful")
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: virtualSwitch.Id,
		},
		NodeType: node.NodeType_switch,
	}, nil
}

func RealMain(serverConfiguration *deployer.Configuration) {
	nsxtClient, err := createNsxtClient(serverConfiguration)
	if err != nil {
		status.New(codes.Internal, fmt.Sprintf("CreateVirtualSwitch: client error (%v)", err))
		return
	}

	listeningAddress, addressError := net.Listen("tcp", serverConfiguration.ServerAddress)
	if addressError != nil {
		log.Fatalf("failed to listen: %v", addressError)
	}

	server := grpc.NewServer()
	node.RegisterNodeServiceServer(server, &nsxtNodeServer{
		Client:        nsxtClient,
		Configuration: serverConfiguration,
	})
	//???
	log.Printf("server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("failed to serve: %v", bindError)
	}
	time.Sleep(3 * time.Second)
}

func main() {
	log.SetPrefix("switcher: ")
	log.SetFlags(0)

	configuration, configurationError := deployer.GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}

	RealMain(configuration)
}
