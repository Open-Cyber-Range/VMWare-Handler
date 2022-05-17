package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	"github.com/ScottHolden/go-vmware-nsxt/manager"
	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const transportZone = "a1facc11-3bad-4b25-885f-906fc5b0ac39" // TZ-Kaarli-Overlay

func createNsxtClient() (nsxtClient *nsxt.APIClient, err error) {
	configuration := nsxt.NewConfiguration()
	configuration.Host = os.Getenv("NSXT_API") // TODO
	configuration.DefaultHeader["Authorization"] = os.Getenv("NSXT_AUTH")
	configuration.Insecure = true

	nsxtClient, err = nsxt.NewAPIClient(configuration)
	if err != nil {
		return
	}
	return
}

func CreateLogicalSwitch(ctx context.Context, nodeDeployment *node.NodeDeployment) (identifier *node.NodeIdentifier, err error) {
	nsxtClient, err := createNsxtClient()
	if err != nil {
		status.New(codes.Internal, fmt.Sprintf("CreateLogicalSwitch: client error (%v)", err))
		return
	}
	newLogicalSwitch := manager.LogicalSwitch{
		TransportZoneId: transportZone,
		DisplayName:     nodeDeployment.GetParameters().GetName(),
		AdminState:      "UP",
		ReplicationMode: "MTEP",
	}
	logicalSwitch, httpResponse, err := nsxtClient.LogicalSwitchingApi.CreateLogicalSwitch(ctx, newLogicalSwitch)
	if err != nil {
		status.New(codes.Internal, fmt.Sprintf("CreateLogicalSwitch: API request error (%v)", err))
		return
	}
	if httpResponse.StatusCode != http.StatusCreated {
		status.New(codes.Internal, fmt.Sprintf("CreateLogicalSwitch: Logical Switch not created (%v)", httpResponse.Status))
		return
	}

	status.New(codes.OK, "Logical Switch creation successful")
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: logicalSwitch.Id,
		},
		NodeType: node.NodeType_switch,
	}, nil
}
