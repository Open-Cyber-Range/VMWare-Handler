package switcher

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	nsxtCommon "github.com/ScottHolden/go-vmware-nsxt/common"
	"github.com/ScottHolden/go-vmware-nsxt/manager"
	deployer "github.com/open-cyber-range/vmware-node-deployer/deployer"
	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func createNsxtConfiguration(serverConfiguration *deployer.Configuration) (nsxtConfiguration *nsxt.Configuration) {
	nsxtConfiguration = nsxt.NewConfiguration()
	nsxtConfiguration.Host = serverConfiguration.NsxtApi
	nsxtConfiguration.DefaultHeader["Authorization"] = serverConfiguration.NsxtAuth
	nsxtConfiguration.Insecure = serverConfiguration.NsxtInsecure
	return
}

func createNsxtClient(serverConfiguration *deployer.Configuration) (nsxtClient *nsxt.APIClient, err error) {
	err = serverConfiguration.ValidateForSwitcher()
	if err == nil {
		nsxtConfiguration := createNsxtConfiguration(serverConfiguration)
		nsxtClient, err = nsxt.NewAPIClient(nsxtConfiguration)
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
	for _, transportNode := range transportZones.Results {
		if strings.EqualFold(transportNode.DisplayName, serverConfiguration.TransportZoneName) {
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
	virtualSwitchDisplayName := nodeDeployment.GetParameters().GetName()
	log.Printf("received request for switch creation: %v\n", virtualSwitchDisplayName)
	transportZoneId, err := findTransportZoneIdByName(ctx, server.Client, server.Configuration)
	if err != nil {
		return
	}
	newVirtualSwitch := manager.LogicalSwitch{
		TransportZoneId: transportZoneId,
		DisplayName:     virtualSwitchDisplayName,
		AdminState:      "UP",
		ReplicationMode: "MTEP",
		Description:     fmt.Sprintf("Created for exercise: %v", nodeDeployment.GetParameters().GetExerciseName()),
		Tags:            []nsxtCommon.Tag{{Scope: "policyPath", Tag: fmt.Sprintf("/infra/segments/%v", virtualSwitchDisplayName)}},
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

	log.Printf("virtual switch created: %v in transport zone: %v\n", virtualSwitch.Id, virtualSwitch.TransportZoneId)
	status.New(codes.OK, "Virtual Switch creation successful")
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: virtualSwitch.Id,
		},
		NodeType: node.NodeType_switch,
	}, nil
}

func delete(ctx context.Context, nsxtClient *nsxt.APIClient, virtualSwitchUuid string) error {
	switchExists, err := virtualSwitchExists(ctx, nsxtClient, virtualSwitchUuid)
	if err != nil {
		return err
	}
	if !switchExists {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("DeleteVirtualSwitch: Switch UUID \" %v \" not found", virtualSwitchUuid))

	} else {
		_, err := nsxtClient.LogicalSwitchingApi.DeleteLogicalSwitch(ctx, virtualSwitchUuid, nil)
		if err != nil {
			status.New(codes.Internal, fmt.Sprintf("DeleteVirtualSwitch: API request error (%v)", err))
			return err
		}
		switchExists, err = virtualSwitchExists(ctx, nsxtClient, virtualSwitchUuid)
		if err != nil {
			return err
		}
		if switchExists {
			return status.Error(codes.Internal, fmt.Sprintf("DeleteVirtualSwitch: Switch UUID \" %v \" was not deleted", virtualSwitchUuid))
		}
	}
	return nil
}

func virtualSwitchExists(ctx context.Context, nsxtClient *nsxt.APIClient, virtualSwitchUuid string) (bool, error) {
	_, httpResponse, err := nsxtClient.LogicalSwitchingApi.GetLogicalSwitch(ctx, virtualSwitchUuid)
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return false, err
	}
	return httpResponse.StatusCode == http.StatusOK, nil
}

func (server *nsxtNodeServer) Delete(ctx context.Context, nodeIdentifier *node.NodeIdentifier) (*emptypb.Empty, error) {
	if *nodeIdentifier.GetNodeType().Enum() == *node.NodeType_switch.Enum() {
		log.Printf("Received switch for deleting: UUID: %v\n", nodeIdentifier.GetIdentifier().GetValue())

		err := delete(ctx, server.Client, nodeIdentifier.GetIdentifier().GetValue())
		if err != nil {
			return nil, err
		}
		log.Printf("virtual switch deleted: %v\n", nodeIdentifier.GetIdentifier().GetValue())
		status.New(codes.OK, "Virtual Switch deletion successful")
		return new(emptypb.Empty), nil
	}
	return nil, status.Error(codes.InvalidArgument, "DeleteVirtualSwitch: Node is not a virtual switch")
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
	log.Printf("server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("failed to serve: %v", bindError)
	}
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
