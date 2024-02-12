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
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type deputyQueryServer struct {
	deputy.UnimplementedDeputyQueryServiceServer
	ServerSpecs *serverSpecs
}

type serverSpecs struct {
	Configuration *library.Configuration
	Storage       *library.Storage[library.PackageContainer]
}

func (server *deputyQueryServer) Create(ctx context.Context, source *common.Source) (*deputy.DeputyCreateResponse, error) {
	packagePath, executorPackage, err := library.GetPackageMetadata(
		source.GetName(),
		source.GetVersion(),
	)
	if err != nil {
		log.Errorf("Error getting package metadata: %v", err)
		return &deputy.DeputyCreateResponse{}, status.Error(codes.Internal, fmt.Sprintf("Error getting package metadata: %v", err))
	}
	filename := executorPackage.GetFilename()
	if filename == "" {
		log.Errorf("Unexpected package file path (is empty)")
		return &deputy.DeputyCreateResponse{}, status.Error(codes.Internal, "Unexpected package file path (is empty)")
	}

	filePath := packagePath + "/" + filename
	htmlPath, checksum, err := library.ConvertMarkdownToHtml(filePath)
	if err != nil {
		log.Errorf("Error converting md to HTML: %v", err)
		return &deputy.DeputyCreateResponse{}, status.Error(codes.Internal, fmt.Sprintf("Error converting md to HTML: %v", err))
	}

	fileMetadata, err := os.Stat(htmlPath)
	if err != nil {
		log.Errorf("Error getting file metadata: %v", err)
		return &deputy.DeputyCreateResponse{}, status.Error(codes.Internal, fmt.Sprintf("Error getting file metadata: %v", err))
	}

	log.Infof("Converted md to HTML: %v, %v, %v bytes", htmlPath, checksum, fileMetadata.Size())

	server.ServerSpecs.Storage.Container = library.PackageContainer{
		Path:     htmlPath,
		Name:     fileMetadata.Name(),
		Size:     fileMetadata.Size(),
		Checksum: checksum,
	}

	packageId := uuid.New().String()
	if err = server.ServerSpecs.Storage.Create(ctx, packageId); err != nil {
		log.Errorf("Error creating package metadata storage: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating package metadata storage: %v", err))
	}

	return &deputy.DeputyCreateResponse{
		Id:       packageId,
		Checksum: checksum,
		Filename: fileMetadata.Name(),
		Size:     fileMetadata.Size(),
	}, nil
}

func (server *deputyQueryServer) Stream(identifier *common.Identifier, stream deputy.DeputyQueryService_StreamServer) error {
	ctx := context.Background()

	packageContainer, err := server.ServerSpecs.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		log.Errorf("Error getting package metadata from storage: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("Error getting package metadata from storage: %v", err))
	}

	file, err := os.Open(packageContainer.Path)
	if err != nil {
		log.Errorf("Error opening package file: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("Error opening package file: %v", err))
	}
	defer file.Close()

	var chunkSize = 1024 * 8
	chunk := make([]byte, chunkSize)
	var readBytes int

	for {
		readBytes, err = file.Read(chunk)
		if err != nil {
			if err != io.EOF {
				log.Errorf("Error reading package file chunk: %v", err)
				return status.Error(codes.Internal, fmt.Sprintf("Error reading package file chunk: %v", err))
			}
			break
		}

		response := &deputy.DeputyStreamResponse{Chunk: chunk[:readBytes]}

		if err := stream.Send(response); err != nil {
			log.Errorf("Error streaming package chunk: %v", err)
			return status.Error(codes.Internal, fmt.Sprintf("Error streaming package chunk: %v", err))
		}
	}
	return nil
}

func (server *deputyQueryServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {

	deputyInfoContainer, err := server.ServerSpecs.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, fmt.Sprintf("Error getting deputyInfoContainer: %v", err))
	}

	log.Debugf("Deleting package file: %v", deputyInfoContainer.Path)

	if err := os.Remove(deputyInfoContainer.Path); err != nil {
		log.Errorf("Error deleting file: %v", err)
		return &emptypb.Empty{}, status.Error(codes.Internal, fmt.Sprintf("Error deleting file: %v", err))
	}

	if err := server.ServerSpecs.Storage.Delete(ctx, identifier.GetValue()); err != nil {
		log.Errorf("Error deleting package metadata from storage: %v", err)
		return &emptypb.Empty{}, status.Error(codes.Internal, fmt.Sprintf("Error deleting package metadata from storage: %v", err))
	}
	return &emptypb.Empty{}, nil
}

func (server *deputyQueryServer) GetPackagesByType(ctx context.Context, query *deputy.GetPackagesQuery) (*deputy.GetPackagesResponse, error) {
	listCommand := exec.Command("deputy", "list", "-t", query.GetPackageType(), "-a")
	output, err := listCommand.CombinedOutput()
	if err != nil {
		log.Errorf("Deputy list command failed, %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v (%v)", string(output), err))
	}

	packageList, err := library.ParseListCommandOutput(output)
	if err != nil {
		log.Errorf("Error parsing deputy list command output, %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error parsing deputy list command output, %v", err))
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
		return &deputy.GetScenarioResponse{}, status.Error(codes.Internal, fmt.Sprintf("Error getting package metadata: %v", err))
	} else if executorPackage.Exercise.FilePath == "" {
		log.Errorf("Unexpected Exercise file path (is empty)")
		return &deputy.GetScenarioResponse{}, status.Error(codes.Internal, "Unexpected Exercise file path (is empty)")
	}

	filePath := packagePath + "/" + executorPackage.Exercise.FilePath
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		log.Errorf("Error reading scenario file: %v", err)
		return &deputy.GetScenarioResponse{}, status.Error(codes.Internal, fmt.Sprintf("Error reading scenario file: %v", err))
	}

	return &deputy.GetScenarioResponse{Sdl: string(fileContents)}, nil
}

func RealMain(configuration *library.Configuration) {
	listeningAddress, addressError := net.Listen("tcp", configuration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
	}

	storage := library.NewStorage[library.PackageContainer](configuration.RedisAddress, configuration.RedisPassword)
	grpcServer := grpc.NewServer()

	serverSpecs := serverSpecs{
		Configuration: configuration,
		Storage:       &storage,
	}
	deputy.RegisterDeputyQueryServiceServer(grpcServer, &deputyQueryServer{ServerSpecs: &serverSpecs})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerType{
		*capability.Capabilities_EventInfo.Enum(),
		*capability.Capabilities_DeputyQuery.Enum(),
	})

	capability.RegisterCapabilityServer(grpcServer, &capabilityServer)

	log.Printf("General listening at %v", listeningAddress.Addr())
	log.Printf("Version: %v", library.Version)

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
