package main

import (
	"context"
	"io"
	"net"
	"os"

	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/google/uuid"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/event"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type eventInfoServer struct {
	event.UnimplementedEventInfoServiceServer
	ServerSpecs *serverSpecs
}

type serverSpecs struct {
	Client        *govmomi.Client
	Configuration *library.Configuration
	Storage       *library.Storage[library.EventInfoContainer]
	MutexPool     *library.MutexPool
}

func (server *eventInfoServer) Create(ctx context.Context, source *common.Source) (*event.EventCreateResponse, error) {

	packagePath, executorPackage, err := library.GetPackageMetadata(
		source.GetName(),
		source.GetVersion(),
	)
	if err != nil {
		log.Errorf("Error getting package metadata: %v", err)
		return &event.EventCreateResponse{}, err
	}

	// Placeholder, will be replaced in the future Deputy release
	filePath := packagePath + "/" + executorPackage.GetAction()
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

func RealMain(configuration *library.Configuration) {
	ctx := context.Background()
	govmomiClient, clientError := configuration.CreateClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	listeningAddress, addressError := net.Listen("tcp", configuration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
	}

	storage := library.NewStorage[library.EventInfoContainer](configuration.RedisAddress, configuration.RedisPassword)
	grpcServer := grpc.NewServer()
	redisClient := goredislib.NewClient(&goredislib.Options{
		Addr:     configuration.RedisAddress,
		Password: configuration.RedisPassword,
	})
	redisPool := goredis.NewPool(redisClient)

	mutexPool, err := library.NewMutexPool(ctx, configuration.Hostname, *redsync.New(redisPool), *redisClient, configuration.Variables)
	if err != nil {
		log.Fatal(err)
	}

	serverSpecs := serverSpecs{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &storage,
		MutexPool:     &mutexPool,
	}
	event.RegisterEventInfoServiceServer(grpcServer, &eventInfoServer{ServerSpecs: &serverSpecs})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_EventInfo.Enum(),
		*capability.Capabilities_DeputyQuery.Enum(),
	})

	capability.RegisterCapabilityServer(grpcServer, &capabilityServer)

	log.Printf("Executor listening at %v", listeningAddress.Addr())

	if bindError := grpcServer.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	configuration, configurationError := library.NewValidator().SetRequireExerciseRootPath(true).GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}
	RealMain(&configuration)
}
