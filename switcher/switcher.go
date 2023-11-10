package main

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/antihax/optional"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	common "github.com/open-cyber-range/vmware-handler/grpc/common"
	switch_grpc "github.com/open-cyber-range/vmware-handler/grpc/switch"
	"github.com/open-cyber-range/vmware-handler/library"
	swagger "github.com/open-cyber-range/vmware-handler/nsx_t_openapi"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

type switchServer struct {
	switch_grpc.UnimplementedSwitchServiceServer
	Configuration Configuration
	APIClient     *swagger.APIClient
}

type DeploySwitch struct {
	DeploymentMessge *switch_grpc.DeploySwitch
	Configuration    Configuration
	APIClient        *swagger.APIClient
}

func (deploySwitch *DeploySwitch) getTransportZone(ctx context.Context) (transportZone *swagger.PolicyTransportZone, err error) {
	segmentApiService := deploySwitch.APIClient.ConnectivityApi
	policyTransportZoneListResult, _, err := segmentApiService.ListTransportZonesForEnforcementPoint(ctx, deploySwitch.Configuration.SiteId,
		"default", &swagger.ConnectivityApiListTransportZonesForEnforcementPointOpts{})
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("getTransportZone: %v", err))
		return nil, err
	}
	for _, transportZone := range policyTransportZoneListResult.Results {
		if deploySwitch.Configuration.TransportZoneName == transportZone.DisplayName {
			return &transportZone, nil
		}
	}
	return nil, status.Error(codes.Internal, fmt.Sprintln("getTransportZone: could not find transportzone"))
}

func (deploySwitch *DeploySwitch) createNetworkSegment(ctx context.Context) (*swagger.Segment, error) {
	transportZone, err := deploySwitch.getTransportZone(ctx)
	if err != nil {
		return nil, err
	}
	var segment = swagger.Segment{
		Id:                library.SanitizeToCompatibleName(deploySwitch.DeploymentMessge.MetaInfo.GetExerciseName() + "_" + deploySwitch.DeploymentMessge.MetaInfo.GetDeploymentName() + "_" + deploySwitch.DeploymentMessge.Switch.GetName()),
		TransportZonePath: transportZone.Path,
	}
	segmentApiService := deploySwitch.APIClient.SegmentsApi
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

func (switchServer *switchServer) deleteInfraSegment(ctx context.Context, virtualSwitchUuid string) (bool, error) {
	segmentApiService := switchServer.APIClient.SegmentsApi
	httpResponse, err := segmentApiService.ForceDeleteInfraSegment(ctx, virtualSwitchUuid, &swagger.SegmentsApiForceDeleteInfraSegmentOpts{Cascade: optional.NewBool(false)})
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return false, err
	}
	return httpResponse.StatusCode == http.StatusOK, nil
}

func (switchServer *switchServer) segmentExists(ctx context.Context, virtualSwitchUuid string) (bool, error) {
	segmentApiService := switchServer.APIClient.SegmentsApi
	_, httpResponse, err := segmentApiService.ReadInfraSegment(ctx, virtualSwitchUuid)
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return false, err
	}
	return httpResponse.StatusCode == http.StatusOK, nil
}

func (switchServer *switchServer) deleteAndVerifyInfraSegment(ctx context.Context, virtualSwitchUuid string) error {
	switchExists, err := switchServer.segmentExists(ctx, virtualSwitchUuid)
	if err != nil {
		return fmt.Errorf("segment check error (%v)", err)
	}
	if !switchExists {
		return fmt.Errorf("switch (UUID \" %v \") not found", virtualSwitchUuid)
	} else {
		_, err = switchServer.deleteInfraSegment(ctx, virtualSwitchUuid)
		if err != nil {
			return fmt.Errorf("API request error (%v)", err)
		}
		switchExists, err = switchServer.segmentExists(ctx, virtualSwitchUuid)
		if err != nil {
			return fmt.Errorf("segment check error (%v)", err)
		}
		if switchExists {
			return fmt.Errorf("switch (UUID \" %v \") was not deleted", virtualSwitchUuid)
		}
	}
	return nil
}

func (server *switchServer) Create(ctx context.Context, switchDeployment *switch_grpc.DeploySwitch) (identifier *common.Identifier, err error) {
	var deploySwitch = DeploySwitch{
		DeploymentMessge: switchDeployment,
		Configuration:    server.Configuration,
		APIClient:        server.APIClient,
	}
	log.Printf("received request for switch creation: %v\n", deploySwitch.DeploymentMessge.Switch.GetName())
	segment, err := deploySwitch.createNetworkSegment(ctx)
	if err != nil {
		log.Printf("virtual segment creation failed: %v", err)
		return
	}
	log.Infof("Virtual segment: %v created in transport zone: %v", segment.Id, segment.TransportZonePath)
	return &common.Identifier{
		Value: segment.Id,
	}, nil
}

func (server *switchServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {

	log.Infof("Received segment for deleting: UUID: %v", identifier.GetValue())

	err := server.deleteAndVerifyInfraSegment(ctx, identifier.GetValue())
	if err != nil {
		log.Errorf("Failed to delete segment (%v)", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to delete segment (%v)", err))
	}
	log.Infof("Deleted segment: %v", identifier.GetValue())
	return new(emptypb.Empty), nil

}

func RealMain(serverConfiguration *Configuration) {
	listeningAddress, addressError := net.Listen("tcp", serverConfiguration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
	}

	server := grpc.NewServer()
	apiClient := createAPIClient(serverConfiguration)
	switch_grpc.RegisterSwitchServiceServer(server, &switchServer{
		APIClient:     apiClient,
		Configuration: *serverConfiguration,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerType{
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
