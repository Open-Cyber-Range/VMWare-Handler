package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/google/uuid"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/condition"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
	"github.com/open-cyber-range/vmware-handler/grpc/inject"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mutexWrapper struct {
	Mutex *redsync.Mutex
}
type featurerServer struct {
	feature.UnimplementedFeatureServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
	Storage       *library.Storage[library.ExecutorContainer]
	Mutex         *mutexWrapper
}

type conditionerServer struct {
	condition.UnimplementedConditionServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
	Storage       *library.Storage[library.ExecutorContainer]
	Mutex         *mutexWrapper
}

type injectServer struct {
	inject.UnimplementedInjectServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
	Storage       *library.Storage[library.ExecutorContainer]
	Mutex         *mutexWrapper
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

type Inject struct {
	Action string     `json:"action,omitempty"`
	Assets [][]string `json:"assets"`
}

func (mutex *mutexWrapper) lock() (err error) {

	successChannel := make(chan bool)
	errorChannel := make(chan error)

	timeout := 30 * 60 * time.Second

	go func() {
		err = mutex.Mutex.Lock()
		if err != nil {
			if strings.HasPrefix(err.Error(), "lock already taken") {
				log.Tracef("Mutex lock taken, trying again")
				time.Sleep(time.Millisecond * (time.Duration(rand.Intn(100))))

				if err = mutex.lock(); err != nil {
					successChannel <- false
					errorChannel <- err
				}
			} else {
				successChannel <- false
				errorChannel <- err
			}
		}
		successChannel <- true
		errorChannel <- nil
	}()
	select {

	case isSuccess := <-successChannel:
		if !isSuccess {
			err := <-errorChannel
			if err != nil {
				return err
			}
		}

	case <-time.After(timeout):
		return status.Error(codes.Internal, fmt.Sprintf("Mutex lock timed out after %v seconds: %v", timeout, err))
	}

	return nil
}

func (mutex *mutexWrapper) unlock() (err error) {
	if isUnlocked, err := mutex.Mutex.Unlock(); !isUnlocked || err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Mutex unlock failed: %v", err))
	}
	return
}

func unmarshalFeature(packageDataMap *map[string]interface{}) (feature Feature, err error) {
	featureInfo := (*packageDataMap)["feature"]
	infoJson, err := json.Marshal(featureInfo)
	if err != nil {
		return Feature{}, err
	}
	if err = json.Unmarshal(infoJson, &feature); err != nil {
		return Feature{}, err
	}
	return
}

func unmarshalCondition(packageDataMap *map[string]interface{}) (condition Condition, err error) {
	conditionInfo := (*packageDataMap)["condition"]
	infoJson, err := json.Marshal(conditionInfo)
	if err != nil {
		return Condition{}, err
	}
	if err = json.Unmarshal(infoJson, &condition); err != nil {
		return Condition{}, err
	}
	return
}

	func unmarshalInject(packageDataMap *map[string]interface{}) (inject Inject, err error) {
		injectInfo := (*packageDataMap)["inject"]
		infoJson, err := json.Marshal(injectInfo)
		if err != nil {
			return Inject{}, err
		}
		if err = json.Unmarshal(infoJson, &inject); err != nil {
			return Inject{}, err
		}
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

func getInjectPackageInfo(injectDeployment *inject.Inject) (packagePath string, inject Inject, err error) {
	packageName := injectDeployment.GetSource().GetName()
	packageVersion := injectDeployment.GetSource().GetVersion()

	packagePath, packageTomlContent, err := getPackageContents(packageName, packageVersion)
	if err != nil {
		return "", Inject{}, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}
	inject, err = unmarshalInject(&packageTomlContent)
	if err != nil {
		return "", Inject{}, status.Error(codes.Internal, fmt.Sprintf("Error getting package info: %v", err))
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
		if err = server.Mutex.lock(); err != nil {
			return nil, err
		}
		packagePath, packageCondition, err := getConditionPackageInfo(conditionDeployment)
		if err != nil {
			return nil, err
		}

		commandAction = packageCondition.Action
		interval = int32(packageCondition.Interval)

		assetFilePaths, err = guestManager.CopyAssetsToVM(ctx, packageCondition.Assets, packagePath)
		if err != nil {
			return nil, err
		}
		if err = library.CleanupTempPackage(packagePath); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error during temp Condition package cleanup (%v)", err))
		}
		if err := server.Mutex.unlock(); err != nil {
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
	if err = server.Storage.Create(ctx, conditionId); err != nil {
		return nil, err
	}

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
		if err = server.Mutex.lock(); err != nil {
			return err
		}
		commandReturnValue, err := guestManager.ExecutePackageAction(ctx, container.Command)
		if err != nil {
			return err
		}
		if err := server.Mutex.unlock(); err != nil {
			return err
		}

		conditionValue, err := strconv.ParseFloat(commandReturnValue, 32)
		if err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Error converting Condition return value to float: %v", err))
		}
		log.Tracef("Condition ID '%v' returned '%v'", identifier.GetValue(), conditionValue)

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

	if err = server.Mutex.lock(); err != nil {
		return new(emptypb.Empty), err
	}
	if err = vmwareClient.DeleteUploadedFiles(ctx, &executorContainer); err != nil {
		return nil, err
	}
	if err := server.Mutex.unlock(); err != nil {
		return new(emptypb.Empty), err
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

	if err = server.Mutex.lock(); err != nil {
		return nil, err
	}

	packagePath, packageFeature, err := getFeaturePackageInfo(featureDeployment)
	if err != nil {
		return nil, err
	}

	featureId := uuid.New().String()

	assetFilePaths, err := guestManager.CopyAssetsToVM(ctx, packageFeature.Assets, packagePath)
	if err != nil {
		return nil, err
	}
	if err = library.CleanupTempPackage(packagePath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during temp Feature package cleanup (%v)", err))
	}
	if err := server.Mutex.unlock(); err != nil {
		return nil, err
	}

	currentDeployment := library.ExecutorContainer{
		VMID:      featureDeployment.GetVirtualMachineId(),
		Auth:      *guestManager.Auth,
		FilePaths: assetFilePaths,
	}

	server.Storage.Container = currentDeployment
	if err = server.Storage.Create(ctx, featureId); err != nil {
		return nil, err
	}

	if err = server.Mutex.lock(); err != nil {
		return nil, err
	}
	var vmLog string
	if packageFeature.Action != "" && featureDeployment.GetFeatureType() == *feature.FeatureType_service.Enum() {
		vmLog, err = guestManager.ExecutePackageAction(ctx, packageFeature.Action)
		if err != nil {
			return nil, err
		}
	}
	if err := server.Mutex.unlock(); err != nil {
		return nil, err
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

	if err = server.Mutex.lock(); err != nil {
		return new(emptypb.Empty), err
	}
	if err = vmwareClient.DeleteUploadedFiles(ctx, &executorContainer); err != nil {
		return nil, err
	}
	if err := server.Mutex.unlock(); err != nil {
		return new(emptypb.Empty), err
	}

	return new(emptypb.Empty), nil
}

func (server *injectServer) Create(ctx context.Context, injectDeployment *inject.Inject) (*common.Identifier, error) {

	auth := &types.NamePasswordAuthentication{
		Username: injectDeployment.GetAccount().GetUsername(),
		Password: injectDeployment.GetAccount().GetPassword(),
	}
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	guestManager, err := vmwareClient.CreateGuestManagers(ctx, injectDeployment.GetVirtualMachineId(), auth)
	if err != nil {
		return nil, err
	}
	if _, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine); err != nil {
		return nil, err
	}

	if err = server.Mutex.lock(); err != nil {
		return nil, err
	}

	packagePath, inject, err := getInjectPackageInfo(injectDeployment)
	if err != nil {
		return nil, err
	}

	assetFilePaths, err := guestManager.CopyAssetsToVM(ctx, inject.Assets, packagePath)
	if err != nil {
		return nil, err
	}
	if err = library.CleanupTempPackage(packagePath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during temp Feature package cleanup (%v)", err))
	}
	if err := server.Mutex.unlock(); err != nil {
		return nil, err
	}

	currentDeployment := library.ExecutorContainer{
		VMID:      injectDeployment.GetVirtualMachineId(),
		Auth:      *guestManager.Auth,
		FilePaths: assetFilePaths,
	}

	injectId := uuid.New().String()
	server.Storage.Container = currentDeployment
	if err = server.Storage.Create(ctx, injectId); err != nil {
		return nil, err
	}

	if err = server.Mutex.lock(); err != nil {
		return nil, err
	}
	if inject.Action != "" {
		_, err = guestManager.ExecutePackageAction(ctx, inject.Action)
		if err != nil {
			return nil, err
		}
	}
	if err := server.Mutex.unlock(); err != nil {
		return nil, err
	}

	return &common.Identifier{
			Value: fmt.Sprintf("%v", injectId),
		},
		nil
}

func (server *injectServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {

	executorContainer, err := server.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return nil, err
	}

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)

	if err = server.Mutex.lock(); err != nil {
		return new(emptypb.Empty), err
	}
	if err = vmwareClient.DeleteUploadedFiles(ctx, &executorContainer); err != nil {
		return nil, err
	}
	if err := server.Mutex.unlock(); err != nil {
		return new(emptypb.Empty), err
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

	grpcServer := grpc.NewServer()

	redisClient := goredislib.NewClient(&goredislib.Options{
		Addr:     configuration.RedisAddress,
		Password: configuration.RedisPassword,
	})
	redisPool := goredis.NewPool(redisClient)
	mutexName := "my-cool-mutex"
	mutex := redsync.New(redisPool).NewMutex(mutexName)

	feature.RegisterFeatureServiceServer(grpcServer, &featurerServer{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &storage,
		Mutex:         &mutexWrapper{Mutex: mutex},
	})

	condition.RegisterConditionServiceServer(grpcServer, &conditionerServer{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &storage,
		Mutex:         &mutexWrapper{Mutex: mutex},
	})

	inject.RegisterInjectServiceServer(grpcServer, &injectServer{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &storage,
		Mutex:         &mutexWrapper{Mutex: mutex},
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Feature.Enum(),
		*capability.Capabilities_Condition.Enum(),
		*capability.Capabilities_Inject.Enum(),
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
