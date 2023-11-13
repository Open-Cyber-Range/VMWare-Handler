package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"

	"github.com/google/uuid"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/deputy"
	"github.com/open-cyber-range/vmware-handler/grpc/event"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type eventInfoServer struct {
	event.UnimplementedEventInfoServiceServer
	ServerSpecs *serverSpecs
}

type deputyQueryServer struct {
	deputy.UnimplementedDeputyQueryServiceServer
	ServerSpecs *serverSpecs
}

type serverSpecs struct {
	Configuration *library.Configuration
	Storage       *library.Storage[library.EventInfoContainer]
}

func (server *eventInfoServer) Create(ctx context.Context, source *common.Source) (*event.EventCreateResponse, error) {
	packagePath, executorPackage, err := library.GetPackageMetadata(
		source.GetName(),
		source.GetVersion(),
	)
	if err != nil {
		log.Errorf("Error getting package metadata: %v", err)
		return &event.EventCreateResponse{}, err
	} else if executorPackage.Event.FilePath == "" {
		log.Errorf("Unexpected Event file path (is empty)")
		return &event.EventCreateResponse{}, fmt.Errorf("unexpected Event file path (is empty)")
	}

	filePath := packagePath + "/" + executorPackage.Event.FilePath
	htmlPath, checksum, err := library.ConvertMarkdownToHtml(filePath)
	if err != nil {
		log.Errorf("Error converting md to HTML: %v", err)
		return &event.EventCreateResponse{}, err
	}

	fileMetadata, err := os.Stat(htmlPath)
	if err != nil {
		log.Errorf("Error getting file metadata: %v", err)
		return &event.EventCreateResponse{}, err
	}

	log.Infof("Converted md to HTML: %v, %v, %v bytes", htmlPath, checksum, fileMetadata.Size())

	server.ServerSpecs.Storage.Container = library.EventInfoContainer{
		Path:     htmlPath,
		Name:     fileMetadata.Name(),
		Size:     fileMetadata.Size(),
		Checksum: checksum,
	}

	eventId := uuid.New().String()
	if err = server.ServerSpecs.Storage.Create(ctx, eventId); err != nil {
		return nil, err
	}

	return &event.EventCreateResponse{
		Id:       eventId,
		Checksum: checksum,
		Filename: fileMetadata.Name(),
		Size:     fileMetadata.Size(),
	}, nil
}

func (server *eventInfoServer) Stream(identifier *common.Identifier, stream event.EventInfoService_StreamServer) error {
	ctx := context.Background()

	eventInfoContainer, err := server.ServerSpecs.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		log.Errorf("Error getting file path from storage: %v", err)
		return err
	}

	file, err := os.Open(eventInfoContainer.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	var chunkSize = 1024 * 8
	chunk := make([]byte, chunkSize)
	var readBytes int

	for {
		readBytes, err = file.Read(chunk)
		if err != nil {
			if err != io.EOF {
				log.Errorf("Error reading chunk: %v", err)
				return err
			}
			break
		}

		response := &event.EventStreamResponse{Chunk: chunk[:readBytes]}

		if err := stream.Send(response); err != nil {
			log.Errorf("Error sending chunk: %v", err)
			return err
		}
	}
	return nil
}

func (server *eventInfoServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	filePath := server.ServerSpecs.Storage.Container.Path
	log.Debugf("Deleting EventInfo file: %v", filePath)

	if err := os.Remove(filePath); err != nil {
		log.Errorf("Error deleting file: %v", err)
		return &emptypb.Empty{}, err
	}

	server.ServerSpecs.Storage.Delete(ctx, identifier.GetValue())
	return &emptypb.Empty{}, nil
}

func (server *deputyQueryServer) GetPackagesByType(ctx context.Context, query *deputy.GetPackagesQuery) (*deputy.GetPackagesResponse, error) {
	listCommand := exec.Command("deputy", "list", "-t", query.GetPackageType())
	output, err := listCommand.CombinedOutput()
	if err != nil {
		log.Errorf("Deputy list command failed, %v", err)
		return nil, fmt.Errorf("%v (%v)", string(output), err)
	}

	packageList, err := library.ParseListCommandOutput(output)
	if err != nil {
		log.Errorf("Error parsing deputy list command output, %v", err)
		return nil, err
	}

	return &deputy.GetPackagesResponse{
			Packages: packageList,
		},
		nil
}

func (server *deputyQueryServer) GetScenario(ctx context.Context, source *common.Source) (*deputy.GetScenarioResponse, error) {
	packagePath, executorPackage, err := library.GetPackageMetadata(
		source.GetName(),
		source.GetVersion(),
	)
	if err != nil {
		log.Errorf("Error getting package metadata: %v", err)
		return &deputy.GetScenarioResponse{}, err
	} else if executorPackage.Exercise.FilePath == "" {
		log.Errorf("Unexpected Exercise file path (is empty)")
		return &deputy.GetScenarioResponse{}, fmt.Errorf("unexpected Exercise file path (is empty)")
	}

	filePath := packagePath + "/" + executorPackage.Exercise.FilePath
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		log.Errorf("Error reading file: %v", err)
		return &deputy.GetScenarioResponse{}, err
	}

	return &deputy.GetScenarioResponse{Sdl: string(fileContents)}, nil
}

func RealMain(configuration *library.Configuration) {
	listeningAddress, addressError := net.Listen("tcp", configuration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
	}

	storage := library.NewStorage[library.EventInfoContainer](configuration.RedisAddress, configuration.RedisPassword)
	grpcServer := grpc.NewServer()

	serverSpecs := serverSpecs{
		Configuration: configuration,
		Storage:       &storage,
	}
	event.RegisterEventInfoServiceServer(grpcServer, &eventInfoServer{ServerSpecs: &serverSpecs})
	deputy.RegisterDeputyQueryServiceServer(grpcServer, &deputyQueryServer{ServerSpecs: &serverSpecs})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerType{
		*capability.Capabilities_EventInfo.Enum(),
		*capability.Capabilities_DeputyQuery.Enum(),
	})

	capability.RegisterCapabilityServer(grpcServer, &capabilityServer)

	log.Printf("General listening at %v", listeningAddress.Addr())

	if bindError := grpcServer.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	validator := library.NewValidator()
	validator.SetRequireRedisConfiguration(true)
	configuration, configurationError := validator.GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}
	RealMain(&configuration)
}
