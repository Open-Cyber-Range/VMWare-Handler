package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type featurerServer struct {
	feature.UnimplementedFeatureServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
	Storage       *library.Storage
}

type Feature struct {
	Type   string     `json:"type"`
	Action string     `json:"action,omitempty"`
	Assets [][]string `json:"assets"`
}

func getFeatureInfo(packegeDataMap *map[string]interface{}) (feature Feature, err error) {
	featureInfo := (*packegeDataMap)["feature"]
	infoJson, err := json.Marshal(featureInfo)
	json.Unmarshal(infoJson, &feature)
	return
}

func (server *featurerServer) Create(ctx context.Context, featureDeployment *feature.Feature) (featureResponse *feature.FeatureResponse, err error) {

	accounts, err := library.Get(ctx, server.Storage.RedisClient, featureDeployment.TemplateId, new([]library.Account))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to Get Feature entry: %v", err))
	}
	log.Infof("Got %v account(s) for current VM", len(*accounts))

	var password string
	for _, account := range *accounts {
		if account.Name == featureDeployment.Username {
			password = account.Password
		}
	}
	auth := &types.NamePasswordAuthentication{
		Username: featureDeployment.Username,
		Password: password,
	}

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	guestManager, err := vmwareClient.CreateGuestManagers(ctx, featureDeployment.VirtualMachineId, auth)
	if err != nil {
		return nil, err
	}

	_, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine)
	if err != nil {
		return nil, err
	}

	packagePath, err := library.DownloadPackage(featureDeployment.GetSource().GetName(), featureDeployment.GetSource().GetVersion())
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}

	packageTomlContent, err := library.GetPackageData(packagePath)
	if err != nil {
		log.Errorf("Failed to get package data, %v", err)
	}

	packageFeature, err := getFeatureInfo(&packageTomlContent)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting package info: %v", err))
	}

	featureId := uuid.New().String()

	currentDeplyoment := library.ExecutorContainer{
		VMID:      featureDeployment.VirtualMachineId,
		Auth:      *guestManager.Auth,
		FilePaths: []string{},
	}

	library.Create(ctx, server.Storage.RedisClient, featureId, currentDeplyoment)

	if err = guestManager.CopyAssetsToVM(ctx, packageFeature.Assets, packagePath, server.Storage, currentDeplyoment, featureId); err != nil {
		return nil, err
	}

	var vmLog string
	if packageFeature.Action != "" && featureDeployment.FeatureType == *feature.FeatureType_service.Enum() {
		vmLog, err = guestManager.ExecutePackageAction(ctx, packageFeature.Action)
		if err != nil {
			return nil, err
		}
	}

	featureResponse = &feature.FeatureResponse{
		Identifier: &common.Identifier{
			Value: fmt.Sprintf("%v", featureId),
		},
		VmLog: vmLog,
	}

	return
}

func (server *featurerServer) deleteDeployedFeature(ctx context.Context, featureContainer *library.ExecutorContainer) error {

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, featureContainer.VMID)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error getting VM by UUID: %v", err))
	}
	operationsManager := guest.NewOperationsManager(virtualMachine.Client(), virtualMachine.Reference())

	fileManager, err := operationsManager.FileManager(ctx)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error creating FileManager: %v", err))
	}

	for i := 0; i < len(featureContainer.FilePaths); i++ {

		targetFile := featureContainer.FilePaths[i]
		if err = fileManager.DeleteFile(ctx, &featureContainer.Auth, string(targetFile)); err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Error deleting file: %v", err))
		}
		log.Infof("Deleted %v", targetFile)

		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {

	featureContainer, err := library.Get(ctx, server.Storage.RedisClient, identifier.GetValue(), new(library.ExecutorContainer))
	if err != nil {
		return nil, err
	}

	if err = server.deleteDeployedFeature(ctx, featureContainer); err != nil {
		return nil, err
	}

	return new(emptypb.Empty), nil
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

	redisClient := library.NewStorage(redis.NewClient(&redis.Options{
		Addr:     configuration.RedisAddress,
		Password: configuration.RedisPassword,
		DB:       0,
	}))
	server := grpc.NewServer()

	feature.RegisterFeatureServiceServer(server, &featurerServer{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &redisClient,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Feature.Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("Executor listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
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
