package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"path"
	"strings"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	nsxtCommon "github.com/ScottHolden/go-vmware-nsxt/common"
	"github.com/ScottHolden/go-vmware-nsxt/manager"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	common "github.com/open-cyber-range/vmware-handler/grpc/common"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func createNsxtClient(serverConfiguration *Configuration) (nsxtClient *nsxt.APIClient, err error) {
	err = serverConfiguration.Validate()
	if err != nil {
		return
	}
	nsxtConfiguration := CreateNsxtConfiguration(serverConfiguration)
	nsxtClient, err = nsxt.NewAPIClient(nsxtConfiguration)
	return
}

func findTransportZoneIdByName(ctx context.Context, nsxtClient *nsxt.APIClient, serverConfiguration Configuration) (string, error) {
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
	Configuration Configuration
}

type capabilityServer struct {
	capability.UnimplementedCapabilityServer
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
		Tags:            []nsxtCommon.Tag{{Scope: "policyPath", Tag: path.Join("/infra/segments", virtualSwitchDisplayName)}},
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

func (server *capabilityServer) GetCapabilities(context.Context, *emptypb.Empty) (*capability.Capabilities, error) {
	status.New(codes.OK, "Switcher reporting for duty")
	return &capability.Capabilities{
		Values: []capability.Capabilities_DeployerTypes{
			*capability.Capabilities_Switch.Enum(),
		},
	}, nil
}

func RealMain(serverConfiguration *Configuration) {
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
		Configuration: *serverConfiguration,
	})

	capability.RegisterCapabilityServer(server, &capabilityServer{})

	log.Printf("server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("failed to serve: %v", bindError)
	}
}

func main() {
	log.SetPrefix("switcher: ")
	log.SetFlags(0)

	configuration, configurationError := GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}

	RealMain(configuration)
}
