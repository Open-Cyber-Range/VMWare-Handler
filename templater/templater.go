package main

import (
	"context"
	"log"
	"net"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
	"github.com/vmware/govmomi"
	"google.golang.org/grpc"
)

// func createNsxtClient(serverConfiguration *Configuration) (nsxtClient *nsxt.APIClient, err error) {
// 	err = serverConfiguration.Validate()
// 	if err != nil {
// 		return
// 	}
// 	nsxtConfiguration := CreateNsxtConfiguration(serverConfiguration)
// 	nsxtClient, err = nsxt.NewAPIClient(nsxtConfiguration)
// 	return
// }

// func findTransportZoneIdByName(ctx context.Context, nsxtClient *nsxt.APIClient, serverConfiguration Configuration) (string, error) {
// 	transportZones, _, err := nsxtClient.NetworkTransportApi.ListTransportZones(ctx, nil)
// 	if err != nil {
// 		status.New(codes.Internal, fmt.Sprintf("CreateVirtualSwitch: ListTransportZones error (%v)", err))
// 		return "", err
// 	}
// 	for _, transportNode := range transportZones.Results {
// 		if strings.EqualFold(transportNode.DisplayName, serverConfiguration.TransportZoneName) {
// 			return transportNode.Id, nil
// 		}
// 	}
// 	return "", status.Error(codes.InvalidArgument, "Transport zone not found")
// }

type templaterServer struct {
	node.UnimplementedNodeServiceServer
	Client        *govmomi.Client
	Configuration library.VMWareConfiguration
}

func RealMain(configuration library.VMWareConfiguration) {
	ctx := context.Background()
	client, clientError := configuration.CreateClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	listeningAddress, addressError := net.Listen("tcp", configuration.ServerAddress)
	if addressError != nil {
		log.Fatalf("failed to listen: %v", addressError)
	}

	server := grpc.NewServer()
	node.RegisterNodeServiceServer(server, &templaterServer{
		Client:        client,
		Configuration: configuration,
	})
	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Template.Enum().Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("failed to serve: %v", bindError)
	}
}

func main() {
	log.SetPrefix("templater: ")
	log.SetFlags(0)

	configuration, configurationError := library.GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}
	log.Println("Hello, templater!")
	RealMain(configuration)
}
