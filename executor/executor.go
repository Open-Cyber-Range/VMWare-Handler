package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

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
	Storage       *library.Storage[library.ExecutorContainer]
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

func (server *featurerServer) getFeaturePackageInfo(featureDeployment *feature.Feature) (packagePath string, feature Feature, err error) {
	packageName := featureDeployment.GetSource().GetName()
	packageVersion := featureDeployment.GetSource().GetVersion()

	packagePath, err = library.DownloadPackage(packageName, packageVersion)
	if err != nil {
		return "", Feature{}, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}

	packageTomlContent, err := library.GetPackageData(packagePath)
	if err != nil {
		log.Errorf("Failed to get package data, %v", err)
	}

	feature, err = getFeatureInfo(&packageTomlContent)
	if err != nil {
		return "", Feature{}, status.Error(codes.Internal, fmt.Sprintf("Error getting package info: %v", err))
	}

	return
}

func (server *featurerServer) Create(ctx context.Context, featureDeployment *feature.Feature) (*feature.FeatureResponse, error) {

	auth := &types.NamePasswordAuthentication{
		Username: featureDeployment.GetAccount().GetUsername(),
		Password: featureDeployment.GetAccount().GetPassword(),
	}
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	guestManager, err := vmwareClient.CreateGuestManagers(ctx, featureDeployment.GetVirtualMachineId(), auth)
	if err != nil {
		return nil, err
	}

	if _, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine); err != nil {
		return nil, err
	}

	packagePath, packageFeature, err := server.getFeaturePackageInfo(featureDeployment)
	if err != nil {
		return nil, err
	}

	featureId := uuid.New().String()

	currentDeplyoment := library.ExecutorContainer{
		VMID:      featureDeployment.GetVirtualMachineId(),
		Auth:      *guestManager.Auth,
		FilePaths: []string{},
	}

	server.Storage.Container = currentDeplyoment
	server.Storage.Create(ctx, featureId)

	if err = guestManager.CopyAssetsToVM(ctx, packageFeature.Assets, packagePath, *server.Storage, featureId); err != nil {
		return nil, err
	}

	var vmLog string
	if packageFeature.Action != "" && featureDeployment.GetFeatureType() == *feature.FeatureType_service.Enum() {
		vmLog, err = guestManager.ExecutePackageAction(ctx, packageFeature.Action)
		if err != nil {
			return nil, err
		}
	}

	return &feature.FeatureResponse{
		Identifier: &common.Identifier{
			Value: fmt.Sprintf("%v", featureId),
		},
		VmLog: vmLog,
	}, nil
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

	featureContainer, err := server.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return nil, err
	}

	if err = server.deleteDeployedFeature(ctx, &featureContainer); err != nil {
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

	storage := library.NewStorage[library.ExecutorContainer](configuration.RedisAddress, configuration.RedisPassword)

	server := grpc.NewServer()

	feature.RegisterFeatureServiceServer(server, &featurerServer{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &storage,
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
