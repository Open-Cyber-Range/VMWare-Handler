package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	common "github.com/open-cyber-range/vmware-handler/grpc/common"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"log"
	"net"
	"net/http"
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

type nsxtNodeServer struct {
	node.UnimplementedNodeServiceServer
	Client        *nsxt.APIClient
	Configuration Configuration
}

type Segment struct {
	DisplayName       string `json:"display_name"`
	Id                string `json:"id,omitempty"`
	TransportZonePath string `json:"transport_zone_path,omitempty"`
}

func createNetworkSegment(nodeDeployment *node.NodeDeployment, serverConfiguration *Configuration) (*Segment, error) {
	var segment = Segment{
		DisplayName: nodeDeployment.GetParameters().GetName(),
	}
	tr := &http.Transport{
		// TODO Insecure https connection here
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	jsonData, err := json.Marshal(segment)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("CreateNetworkSegment: JSON Marshal (%v)", err))
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPut, "https://"+serverConfiguration.NsxtApi+"/policy/api/v1/infra/segments/"+segment.DisplayName, bytes.NewBuffer(jsonData))
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("CreateNetworkSegment: Request creation (%v)", err))
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %v", serverConfiguration.NsxtAuth))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := client.Do(req)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("CreateNetworkSegment: API request (%v)", err))
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		err = status.Error(codes.Internal, fmt.Sprintf("CreateNetworkSegment: Virtual Switch not created (%v)", response.Status))
		return nil, err
	}

	var segmentResponse Segment
	bytearray, _ := io.ReadAll(response.Body)
	_ = json.Unmarshal(bytearray, &segmentResponse)
	return &segmentResponse, nil
}

func (server *nsxtNodeServer) Create(ctx context.Context, nodeDeployment *node.NodeDeployment) (identifier *node.NodeIdentifier, err error) {
	virtualSwitchDisplayName := nodeDeployment.GetParameters().GetName()
	log.Printf("received request for switch creation: %v\n", virtualSwitchDisplayName)
	segment, err := createNetworkSegment(nodeDeployment, &server.Configuration)
	if err != nil {
		log.Printf("virtual segment creation failed: %v", err)
		return
	}
	log.Printf("virtual segment created: %v in transport zone: %v\n", segment.Id, segment.TransportZonePath)
	status.New(codes.OK, "Virtual Switch creation successful")
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: segment.Id,
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

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Switch.Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

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
