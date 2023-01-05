package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/condition"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
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

type conditionerServer struct {
	condition.UnimplementedConditionServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
	Storage       *library.Storage[library.ExecutorContainer]
}

type Feature struct {
	Type   string     `json:"type"`
	Action string     `json:"action,omitempty"`
	Assets [][]string `json:"assets"`
}

type Condition struct {
	Interval uint32     `json:"interval,omitempty"`
	Action   string     `json:"action,omitempty"`
	Assets   [][]string `json:"assets"`
}

func unmarshalFeature(packegeDataMap *map[string]interface{}) (feature Feature, err error) {
	featureInfo := (*packegeDataMap)["feature"]
	infoJson, err := json.Marshal(featureInfo)
	json.Unmarshal(infoJson, &feature)
	return
}

func unmarshalCondition(packegeDataMap *map[string]interface{}) (condition Condition, err error) {
	conditionInfo := (*packegeDataMap)["condition"]
	infoJson, err := json.Marshal(conditionInfo)
	json.Unmarshal(infoJson, &condition)
	return
}

func getPackageContents(packageName string, packageVersion string) (packagePath string, packageTomlContent map[string]interface{}, err error) {
	packagePath, err = library.DownloadPackage(packageName, packageVersion)
	if err != nil {
		return "", nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}

	packageTomlContent, err = library.GetPackageData(packagePath)
	if err != nil {
		log.Errorf("Failed to get package data, %v", err)
	}

	return
}

func getFeaturePackageInfo(featureDeployment *feature.Feature) (packagePath string, feature Feature, err error) {
	packageName := featureDeployment.GetSource().GetName()
	packageVersion := featureDeployment.GetSource().GetVersion()

	packagePath, packageTomlContent, err := getPackageContents(packageName, packageVersion)
	if err != nil {
		return "", Feature{}, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}

	feature, err = unmarshalFeature(&packageTomlContent)
	if err != nil {
		return "", Feature{}, status.Error(codes.Internal, fmt.Sprintf("Error getting package info: %v", err))
	}

	return
}

func getConditionPackageInfo(conditionDeployment *condition.Condition) (packagePath string, condition Condition, err error) {
	packageName := conditionDeployment.GetSource().GetName()
	packageVersion := conditionDeployment.GetSource().GetVersion()

	packagePath, packageTomlContent, err := getPackageContents(packageName, packageVersion)
	if err != nil {
		return "", Condition{}, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}

	condition, err = unmarshalCondition(&packageTomlContent)
	if err != nil {
		return "", Condition{}, status.Error(codes.Internal, fmt.Sprintf("Error getting package info: %v", err))
	}

	return
}

func (server *conditionerServer) Create(ctx context.Context, conditionDeployment *condition.Condition) (*common.Identifier, error) {
	var isSourceDeployment bool

	if conditionDeployment.Source != nil && conditionDeployment.Command != "" {
		return nil, status.Error(codes.Internal, fmt.Sprintln("Conflicting deployment info - A Condition deployment cannot have both `Command` and `Source`"))
	} else if conditionDeployment.Source != nil && conditionDeployment.Command == "" {
		isSourceDeployment = true
	}

	auth := &types.NamePasswordAuthentication{
		Username: conditionDeployment.GetAccount().GetUsername(),
		Password: conditionDeployment.GetAccount().GetPassword(),
	}
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	guestManager, err := vmwareClient.CreateGuestManagers(ctx, conditionDeployment.GetVirtualMachineId(), auth)
	if err != nil {
		return nil, err
	}

	if _, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine); err != nil {
		return nil, err
	}

	conditionId := uuid.New().String()

	var commandAction string
	var interval int32
	var assetFilePaths []string

	if isSourceDeployment {
		packagePath, packageCondition, err := getConditionPackageInfo(conditionDeployment)
		if err != nil {
			return nil, err
		}

		commandAction = packageCondition.Action
		interval = int32(packageCondition.Interval)

		assetFilePaths, err = guestManager.CopyAssetsToVM(ctx, packageCondition.Assets, packagePath, conditionId)
		if err != nil {
			return nil, err
		}

	} else {
		commandAction = conditionDeployment.GetCommand()
		interval = conditionDeployment.GetInterval()
		assetFilePaths = append(assetFilePaths, commandAction)

	}

	server.Storage.Container = library.ExecutorContainer{
		VMID:      conditionDeployment.GetVirtualMachineId(),
		Auth:      *guestManager.Auth,
		FilePaths: assetFilePaths,
		Command:   commandAction,
		Interval:  interval,
	}

	server.Storage.Create(ctx, conditionId)

	return &common.Identifier{Value: conditionId}, nil
}

func (server *conditionerServer) Stream(identifier *common.Identifier, stream condition.ConditionService_StreamServer) error {
	ctx := context.Background()

	container, err := server.Storage.Get(context.Background(), identifier.GetValue())
	if err != nil {
		return err
	}

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	guestManager, err := vmwareClient.CreateGuestManagers(ctx, container.VMID, &container.Auth)
	if err != nil {
		return err
	}

	if _, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine); err != nil {
		return err
	}

	for {
		commandReturnValue, err := guestManager.ExecutePackageAction(ctx, container.Command)
		if err != nil {
			return err
		}

		var conditionValue float64
		if commandReturnValue != "" {
			conditionValue, err = strconv.ParseFloat(commandReturnValue, 32)
			if err != nil {
				return status.Error(codes.Internal, fmt.Sprintf("Error converting PackageActionValue: %v", err))
			}
		}
		log.Infof("Condition return value '%v'", commandReturnValue)

		err = stream.Send(&condition.ConditionStreamResponse{
			Response:           identifier.GetValue(),
			CommandReturnValue: float32(conditionValue),
		})
		if err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Error sending Condition stream: %v", err))
		}

		time.Sleep(time.Second * time.Duration(container.Interval))
	}
}

func (server *conditionerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {

	executorContainer, err := server.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return nil, err
	}

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	if err = vmwareClient.DeleteDeployedFiles(ctx, &executorContainer); err != nil {
		return nil, err
	}

	return new(emptypb.Empty), nil
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

	packagePath, packageFeature, err := getFeaturePackageInfo(featureDeployment)
	if err != nil {
		return nil, err
	}

	featureId := uuid.New().String()

	assetFilePaths, err := guestManager.CopyAssetsToVM(ctx, packageFeature.Assets, packagePath, featureId)
	if err != nil {
		return nil, err
	}

	currentDeployment := library.ExecutorContainer{
		VMID:      featureDeployment.GetVirtualMachineId(),
		Auth:      *guestManager.Auth,
		FilePaths: assetFilePaths,
	}

	server.Storage.Container = currentDeployment
	server.Storage.Create(ctx, featureId)

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

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {

	executorContainer, err := server.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return nil, err
	}

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	if err = vmwareClient.DeleteDeployedFiles(ctx, &executorContainer); err != nil {
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

	condition.RegisterConditionServiceServer(server, &conditionerServer{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &storage,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Feature.Enum(),
		*capability.Capabilities_Condition.Enum(),
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
