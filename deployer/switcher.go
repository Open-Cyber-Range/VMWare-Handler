package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	nsxtCommon "github.com/ScottHolden/go-vmware-nsxt/common"
	"github.com/ScottHolden/go-vmware-nsxt/manager"
	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func createNsxtClient(serverConfiguration *Configuration) (nsxtClient *nsxt.APIClient, err error) {
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

func findTransportZoneIdByName(ctx context.Context, nsxtClient *nsxt.APIClient, serverConfiguration *Configuration) (string, error) {
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

func CreateVirtualSwitch(ctx context.Context, serverConfiguration *Configuration, nodeDeployment *node.NodeDeployment) (identifier *node.NodeIdentifier, err error) {
	nsxtClient, err := createNsxtClient(serverConfiguration)
	if err != nil {
		status.New(codes.Internal, fmt.Sprintf("CreateVirtualSwitch: client error (%v)", err))
		return
	}
	transportZoneId, err := findTransportZoneIdByName(ctx, nsxtClient, serverConfiguration)
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
	virtualSwitch, httpResponse, err := nsxtClient.LogicalSwitchingApi.CreateLogicalSwitch(ctx, newVirtualSwitch)
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
