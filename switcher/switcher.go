package main

import (
	"context"
	"fmt"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	common "github.com/open-cyber-range/vmware-handler/grpc/common"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
	swagger "github.com/open-cyber-range/vmware-handler/nsx_t_openapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"net/http"
)

func createAPIClient(serverConfiguration *Configuration) (apiClient *swagger.APIClient) {
	err := serverConfiguration.Validate()
	if err != nil {
		return
	}
	configuration := CreateAPIConfiguration(serverConfiguration)
	apiClient = swagger.NewAPIClient(configuration)
	return
}

type nsxtNodeServer struct {
	node.UnimplementedNodeServiceServer
	Configuration Configuration
	APIClient     *swagger.APIClient
}

func segmentExists(ctx context.Context, server *nsxtNodeServer, virtualSwitchUuid string) (bool, error) {
	segmentApiService := server.APIClient.SegmentsApi
	_, httpResponse, err := segmentApiService.ReadInfraSegment(ctx, virtualSwitchUuid)
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return false, err
	}
	return httpResponse.StatusCode == http.StatusOK, nil
}

func getTransportZone(ctx context.Context, server *nsxtNodeServer) (transportZone *swagger.PolicyTransportZone, err error) {
	segmentApiService := server.APIClient.ConnectivityApi
	policyTransportZoneListResult, _, err := segmentApiService.ListTransportZonesForEnforcementPoint(ctx, server.Configuration.SiteId,
		"default", &swagger.ConnectivityApiListTransportZonesForEnforcementPointOpts{})
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("getTransportZone: %v", err))
		return nil, err
	}
	for _, transportZone := range policyTransportZoneListResult.Results {
		if server.Configuration.TransportZoneName == transportZone.DisplayName {
			return &transportZone, nil
		}
	}
	return nil, status.Error(codes.Internal, fmt.Sprintf("getTransportZone: could not find transportzone"))
}

func createNetworkSegment(ctx context.Context, nodeDeployment *node.NodeDeployment, server *nsxtNodeServer) (*swagger.Segment, error) {
	transportZone, err := getTransportZone(ctx, server)
	if err != nil {
		return nil, err
	}
	var segment = swagger.Segment{
		Id: nodeDeployment.GetParameters().GetName() + "_" +
			nodeDeployment.GetParameters().GetExerciseName() + "_" + transportZone.Id,
		TransportZonePath: transportZone.Path,
	}
	segmentApiService := server.APIClient.SegmentsApi
	segmentResponse, httpResponse, err := segmentApiService.CreateOrReplaceInfraSegment(ctx, segment.Id, segment)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("CreateSegment: API request (%v)", err))
		return nil, err
	}
	if httpResponse.StatusCode != http.StatusOK {
		err = status.Error(codes.Internal, fmt.Sprintf("CreateSegment: Segment not created (%v)", httpResponse.Status))
		return nil, err
	}
	return &segmentResponse, nil
}

func deleteInfraSegment(ctx context.Context, server *nsxtNodeServer, virtualSwitchUuid string) (bool, error) {
	segmentApiService := server.APIClient.SegmentsApi
	httpResponse, err := segmentApiService.DeleteInfraSegment(ctx, virtualSwitchUuid)
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return false, err
	}
	return httpResponse.StatusCode == http.StatusOK, nil
}

func (server *nsxtNodeServer) Create(ctx context.Context, nodeDeployment *node.NodeDeployment) (identifier *node.NodeIdentifier, err error) {
	virtualSwitchDisplayName := nodeDeployment.GetParameters().GetName()
	log.Printf("received request for switch creation: %v\n", virtualSwitchDisplayName)
	segment, err := createNetworkSegment(ctx, nodeDeployment, server)
	if err != nil {
		log.Printf("virtual segment creation failed: %v", err)
		return
	}
	log.Infof("Virtual segment: %v created in transport zone: %v", segment.Id, segment.TransportZonePath)
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: segment.Id,
		},
		NodeType: node.NodeType_switch,
	}, nil
}

func deleteAndVerifyInfraSegment(ctx context.Context, virtualSwitchUuid string, server *nsxtNodeServer) error {
	switchExists, err := segmentExists(ctx, server, virtualSwitchUuid)
	if err != nil {
		return fmt.Errorf("segment check error (%v)", err)
	}
	if !switchExists {
		return fmt.Errorf("switch (UUID \" %v \") not found", virtualSwitchUuid)
	} else {
		_, err = deleteInfraSegment(ctx, server, virtualSwitchUuid)
		if err != nil {
			return fmt.Errorf("API request error (%v)", err)
		}
		switchExists, err = segmentExists(ctx, server, virtualSwitchUuid)
		if err != nil {
			return fmt.Errorf("segment check error (%v)", err)
		}
		if switchExists {
			return fmt.Errorf("switch (UUID \" %v \") was not deleted", virtualSwitchUuid)
		}
	}
	return nil
}

func (server *nsxtNodeServer) Delete(ctx context.Context, nodeIdentifier *node.NodeIdentifier) (*emptypb.Empty, error) {
	if *nodeIdentifier.GetNodeType().Enum() == *node.NodeType_switch.Enum() {
		log.Infof("Received segment for deleting: UUID: %v", nodeIdentifier.GetIdentifier().GetValue())

		err := deleteAndVerifyInfraSegment(ctx, nodeIdentifier.GetIdentifier().GetValue(), server)
		if err != nil {
			log.Errorf("Failed to delete segment (%v)", err)
			return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to delete segment (%v)", err))
		}
		log.Infof("Deleted segment: %v", nodeIdentifier.GetIdentifier().GetValue())
		return new(emptypb.Empty), nil
	}
	return nil, status.Error(codes.InvalidArgument, "Node is not a virtual switch")
}

func RealMain(serverConfiguration *Configuration) {
	listeningAddress, addressError := net.Listen("tcp", serverConfiguration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
	}

	server := grpc.NewServer()
	apiClient := createAPIClient(serverConfiguration)
	node.RegisterNodeServiceServer(server, &nsxtNodeServer{
		APIClient:     apiClient,
		Configuration: *serverConfiguration,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Switch.Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Infof("Switcher listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	configuration, configurationError := GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}
	RealMain(configuration)
}
