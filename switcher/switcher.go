package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	nsxt "github.com/ScottHolden/go-vmware-nsxt"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	common "github.com/open-cyber-range/vmware-handler/grpc/common"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
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

func segmentExists(serverConfiguration *Configuration, virtualSwitchUuid string) (bool, error) {
	// TODO use OpenAPI for the next 3 following requests
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest(http.MethodGet, "https://"+serverConfiguration.NsxtApi+"/policy/api/v1/infra/segments/"+virtualSwitchUuid, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %v", serverConfiguration.NsxtAuth))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpResponse, err := client.Do(req)
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return false, err
	}
	return httpResponse.StatusCode == http.StatusOK, nil
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
		return nil, fmt.Errorf("JSON Marshal error (%v)", err)
	}
	req, _ := http.NewRequest(http.MethodPut, "https://"+serverConfiguration.NsxtApi+"/policy/api/v1/infra/segments/"+segment.DisplayName, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %v", serverConfiguration.NsxtAuth))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request error (%v)", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("segment not created (%v)", response.Status)
	}

	var segmentResponse Segment
	bytearray, _ := io.ReadAll(response.Body)
	_ = json.Unmarshal(bytearray, &segmentResponse)
	return &segmentResponse, nil
}

func deleteInfraSegment(serverConfiguration *Configuration, virtualSwitchUuid string) (bool, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest(http.MethodDelete, "https://"+serverConfiguration.NsxtApi+"/policy/api/v1/infra/segments/"+virtualSwitchUuid, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %v", serverConfiguration.NsxtAuth))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpResponse, err := client.Do(req)
	if err != nil && httpResponse.StatusCode != http.StatusNotFound {
		return false, err
	}
	return httpResponse.StatusCode == http.StatusOK, nil
}

func (server *nsxtNodeServer) Create(ctx context.Context, nodeDeployment *node.NodeDeployment) (identifier *node.NodeIdentifier, err error) {
	virtualSwitchDisplayName := nodeDeployment.GetParameters().GetName()
	log.Printf("Received request for switch: %v creation", virtualSwitchDisplayName)
	segment, segmentError := createNetworkSegment(nodeDeployment, &server.Configuration)
	if segmentError != nil {
		log.Printf("Virtual segment creation failed (%v)", segmentError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Virtual segment creation failed (%v)", segmentError))
	}
	log.Printf("Virtual segment: %v created in transport zone: %v", segment.Id, segment.TransportZonePath)
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: segment.Id,
		},
		NodeType: node.NodeType_switch,
	}, nil
}

func delete(virtualSwitchUuid string, server *nsxtNodeServer) error {
	switchExists, err := segmentExists(&server.Configuration, virtualSwitchUuid)
	if err != nil {
		return fmt.Errorf("segment check error (%v)", err)
	}
	if !switchExists {
		return fmt.Errorf("switch (UUID \" %v \") not found", virtualSwitchUuid)
	} else {
		_, err = deleteInfraSegment(&server.Configuration, virtualSwitchUuid)
		if err != nil {
			return fmt.Errorf("API request error (%v)", err)
		}
		switchExists, err = segmentExists(&server.Configuration, virtualSwitchUuid)
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
		log.Printf("Received segment for deleting: UUID: %v", nodeIdentifier.GetIdentifier().GetValue())

		err := delete(nodeIdentifier.GetIdentifier().GetValue(), server)
		if err != nil {
			log.Printf("Failed to delete segment (%v)", err)
			return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to delete segment (%v)", err))
		}
		log.Printf("Deleted segment: %v", nodeIdentifier.GetIdentifier().GetValue())
		return new(emptypb.Empty), nil
	}
	return nil, status.Error(codes.InvalidArgument, "Node is not a virtual switch")
}

func RealMain(serverConfiguration *Configuration) {
	nsxtClient, nsxtClientError := createNsxtClient(serverConfiguration)
	if nsxtClientError != nil {
		log.Fatalf("Client error (%v)", nsxtClientError)
	}

	listeningAddress, addressError := net.Listen("tcp", serverConfiguration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
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

	log.Printf("Server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	configuration, configurationError := GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}
	log.Println("Switcher has started")
	RealMain(configuration)
}
