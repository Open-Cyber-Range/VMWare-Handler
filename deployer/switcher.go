package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	nsxtCommon "github.com/ScottHolden/go-vmware-nsxt/common"
	"github.com/ScottHolden/go-vmware-nsxt/manager"
	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const transportZoneKOverlay = "a1facc11-3bad-4b25-885f-906fc5b0ac39"

func createNsxtClient() (nsxtClient *nsxt.APIClient, err error) {
	configuration := nsxt.NewConfiguration()
	configuration.Host = os.Getenv("NSXT_API")
	configuration.DefaultHeader["Authorization"] = os.Getenv("NSXT_AUTH")
	configuration.Insecure = true

	nsxtClient, err = nsxt.NewAPIClient(configuration)
	if err != nil {
		return
	}
	return
}

func CreateVirtualSwitch(ctx context.Context, nodeDeployment *node.NodeDeployment) (identifier *node.NodeIdentifier, err error) {
	nsxtClient, err := createNsxtClient()
	if err != nil {
		status.New(codes.Internal, fmt.Sprintf("CreateVirtualSwitch: client error (%v)", err))
		return
	}
	newVogicalSwitch := manager.LogicalSwitch{
		TransportZoneId: transportZoneKOverlay,
		DisplayName:     nodeDeployment.GetParameters().GetName(),
		AdminState:      "UP",
		ReplicationMode: "MTEP",
		Description:     fmt.Sprintf("Created for exercise: %v", nodeDeployment.GetParameters().GetExerciseName()),
		Tags:            []nsxtCommon.Tag{{Scope: "policyPath", Tag: fmt.Sprintf("/infra/segments/%v", nodeDeployment.GetParameters().GetName())}},
	}
	virtualSwitch, httpResponse, err := nsxtClient.LogicalSwitchingApi.CreateLogicalSwitch(ctx, newVogicalSwitch)
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
