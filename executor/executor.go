package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
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

type featurerServer struct {
	feature.UnimplementedFeatureServiceServer
	ServerSpecs *serverSpecs
}

type conditionerServer struct {
	condition.UnimplementedConditionServiceServer
	ServerSpecs *serverSpecs
}

type injectServer struct {
	inject.UnimplementedInjectServiceServer
	ServerSpecs *serverSpecs
}

type serverSpecs struct {
	Client        *govmomi.Client
	Configuration *library.Configuration
	Storage       *library.Storage[library.ExecutorContainer]
	MutexPool     *library.MutexPool
}

type vmWareTarget struct {
	VmID          string
	VmAccount     *library.Account
	PackageSource *common.Source
	Environment   []string
}

func (server *conditionerServer) createCondition(ctx context.Context, conditionDeployment *condition.Condition, guestManager library.GuestManager) (commandAction string, interval int32, assetFilePaths []string, err error) {
	var isSourceDeployment bool

	if conditionDeployment.Source != nil && conditionDeployment.Command != "" {
		log.Errorf("Conflicting deployment info - A Condition deployment cannot have both `Command` and `Source`")
		return "", 0, nil, status.Error(codes.Internal, fmt.Sprintln("Conflicting deployment info - A Condition deployment cannot have both `Command` and `Source`"))
	} else if conditionDeployment.Source != nil && conditionDeployment.Command == "" {
		isSourceDeployment = true
	}

	if isSourceDeployment {
		packagePath, packageMetadata, err := library.GetPackageMetadata(
			conditionDeployment.GetSource().GetName(),
			conditionDeployment.GetSource().GetVersion(),
		)
		if err != nil {
			log.Errorf("Error getting Condition package metadata: %v", err)
			return "", 0, nil, err
		}

		commandAction = packageMetadata.Condition.Action
		interval = int32(packageMetadata.Condition.Interval)

		assetFilePaths, err = guestManager.CopyAssetsToVM(ctx, packageMetadata.PackageBody.Assets, packagePath)
		if err != nil {
			log.Errorf("Error copying Condition assets to VM: %v", err)
			return "", 0, nil, err
		}
		if err = library.CleanupTempPackage(packagePath); err != nil {
			log.Errorf("Error during temp Condition package cleanup: %v", err)
			return "", 0, nil, status.Error(codes.Internal, fmt.Sprintf("Error during temp Condition package cleanup (%v)", err))
		}
	} else {
		commandAction = conditionDeployment.GetCommand()
		interval = conditionDeployment.GetInterval()
		assetFilePaths = append(assetFilePaths, commandAction)
	}
	return commandAction, interval, assetFilePaths, nil
}

func (server *conditionerServer) Create(ctx context.Context, conditionDeployment *condition.Condition) (*common.Identifier, error) {
	vmAuthentication := &types.NamePasswordAuthentication{
		Username: conditionDeployment.GetAccount().GetUsername(),
		Password: conditionDeployment.GetAccount().GetPassword(),
	}
	vmwareClient, loginError := library.NewVMWareClient(ctx, server.ServerSpecs.Client, *server.ServerSpecs.Configuration)
	if loginError != nil {
		return nil, loginError
	}
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, conditionDeployment.GetVirtualMachineId(), vmAuthentication)
	if err != nil {
		log.Errorf("Error creating Condition: %v", err)
		return nil, err
	}
	if err = library.AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, server.ServerSpecs.Configuration.Variables); err != nil {
		log.Errorf("Error awaiting VMTools to come online: %v", err)
		return nil, err
	}
	mutex, err := server.ServerSpecs.MutexPool.GetMutex(ctx, conditionDeployment.GetVirtualMachineId())
	if err != nil {
		return nil, err
	}
	if err = mutex.Lock(ctx); err != nil {
		return nil, err
	}
	commandAction, interval, assetFilePaths, createErr := server.createCondition(ctx, conditionDeployment, *guestManager)
	if err := mutex.Unlock(ctx); err != nil {
		return nil, err
	}
	if createErr != nil {
		return nil, createErr
	}
	server.ServerSpecs.Storage.Container = library.ExecutorContainer{
		VMID:        conditionDeployment.GetVirtualMachineId(),
		Auth:        *guestManager.Auth,
		FilePaths:   assetFilePaths,
		Command:     commandAction,
		Interval:    interval,
		Environment: conditionDeployment.GetEnvironment(),
	}
	conditionId := uuid.New().String()
	if err = server.ServerSpecs.Storage.Create(ctx, conditionId); err != nil {
		return nil, err
	}

	return &common.Identifier{Value: conditionId}, nil
}

func (server *conditionerServer) Stream(identifier *common.Identifier, stream condition.ConditionService_StreamServer) error {
	ctx := context.Background()
	container, err := server.ServerSpecs.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return err
	}

	vmwareClient, loginError := library.NewVMWareClient(ctx, server.ServerSpecs.Client, *server.ServerSpecs.Configuration)
	if loginError != nil {
		return loginError
	}
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, container.VMID, &container.Auth)
	if err != nil {
		return err
	}
	for {
		if err = library.AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, server.ServerSpecs.Configuration.Variables); err != nil {
			return err
		}
		mutex, err := server.ServerSpecs.MutexPool.GetMutex(ctx, container.VMID)
		if err != nil {
			return err
		}
		if err = mutex.Lock(ctx); err != nil {
			return err
		}
		commandReturnValue, executeErr := guestManager.ExecutePackageAction(ctx, container.Command, container.Environment)
		if err := mutex.Unlock(ctx); err != nil {
			return err
		}
		if executeErr != nil {
			return executeErr
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
	return uninstallPackage(ctx, server.ServerSpecs, identifier)
}

func (server *featurerServer) Create(ctx context.Context, featureDeployment *feature.Feature) (*common.ExecutorResponse, error) {
	vmWareTarget := vmWareTarget{
		VmAccount: &library.Account{
			Name:     featureDeployment.GetAccount().GetUsername(),
			Password: featureDeployment.GetAccount().GetPassword(),
		},
		PackageSource: featureDeployment.GetSource(),
		VmID:          featureDeployment.GetVirtualMachineId(),
		Environment:   featureDeployment.GetEnvironment(),
	}
	mutex, err := server.ServerSpecs.MutexPool.GetMutex(ctx, featureDeployment.GetVirtualMachineId())
	if err != nil {
		return nil, err
	}
	if err = mutex.Lock(ctx); err != nil {
		return nil, err
	}
	executorResponse, installErr := installPackage(ctx, &vmWareTarget, server.ServerSpecs)
	if err := mutex.Unlock(ctx); err != nil {
		return nil, err
	}
	if installErr != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during Feature Create: %v", installErr))
	}

	return executorResponse, nil
}

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	return uninstallPackage(ctx, server.ServerSpecs, identifier)
}

func (server *injectServer) Create(ctx context.Context, injectDeployment *inject.Inject) (*common.ExecutorResponse, error) {
	vmWareTarget := vmWareTarget{
		VmID: injectDeployment.GetVirtualMachineId(),
		VmAccount: &library.Account{
			Name:     injectDeployment.GetAccount().GetUsername(),
			Password: injectDeployment.GetAccount().GetPassword(),
		},
		PackageSource: injectDeployment.GetSource(),
		Environment:   injectDeployment.GetEnvironment(),
	}
	mutex, err := server.ServerSpecs.MutexPool.GetMutex(ctx, injectDeployment.GetVirtualMachineId())
	if err != nil {
		return nil, err
	}
	if err = mutex.Lock(ctx); err != nil {
		return nil, err
	}
	executorResponse, installErr := installPackage(ctx, &vmWareTarget, server.ServerSpecs)
	if err := mutex.Unlock(ctx); err != nil {
		return nil, err
	}
	if installErr != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during Inject Create: %v", installErr))
	}

	return executorResponse, nil
}

func (server *injectServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	return uninstallPackage(ctx, server.ServerSpecs, identifier)
}

func installPackage(ctx context.Context, vmWareTarget *vmWareTarget, serverSpecs *serverSpecs) (*common.ExecutorResponse, error) {
	vmAuthentication := &types.NamePasswordAuthentication{
		Username: vmWareTarget.VmAccount.Name,
		Password: vmWareTarget.VmAccount.Password,
	}
	vmwareClient, loginError := library.NewVMWareClient(ctx, serverSpecs.Client, *serverSpecs.Configuration)
	if loginError != nil {
		return nil, loginError
	}
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, vmWareTarget.VmID, vmAuthentication)
	if err != nil {
		return nil, err
	}
	packageMetadata, assetFilePaths, err := guestManager.UploadPackageContents(ctx, vmWareTarget.PackageSource)
	if err != nil {
		return nil, err
	}
	vmPackageUuid := uuid.New().String()
	serverSpecs.Storage.Container = library.ExecutorContainer{
		VMID:        vmWareTarget.VmID,
		Auth:        *guestManager.Auth,
		FilePaths:   assetFilePaths,
		Environment: vmWareTarget.Environment,
	}
	if err = serverSpecs.Storage.Create(ctx, vmPackageUuid); err != nil {
		return nil, err
	}
	packageAction := packageMetadata.GetAction()
	var vmLog string
	if packageAction != "" {
		vmLog, err = guestManager.ExecutePackageAction(ctx, packageAction, vmWareTarget.Environment)
		if err != nil {
			return nil, err
		}
	}

	if packageMetadata.Feature.Restarts || packageMetadata.Inject.Restarts {
		log.Infof("Restarting VM %v as requested by package %v", guestManager.VirtualMachine.UUID(ctx), packageMetadata.PackageBody.Name)
		if err = guestManager.Reboot(ctx); err != nil {
			log.Errorf("Failed to reboot VM %v: %v", guestManager.VirtualMachine.UUID(ctx), err)
			return nil, err
		}
	}

	return &common.ExecutorResponse{
		Identifier: &common.Identifier{
			Value: fmt.Sprintf("%v", vmPackageUuid),
		},
		VmLog: vmLog,
	}, nil
}

func uninstallPackage(ctx context.Context, serverSpecs *serverSpecs, identifier *common.Identifier) (*emptypb.Empty, error) {
	executorContainer, err := serverSpecs.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return new(emptypb.Empty), err
	}

	vmwareClient, loginError := library.NewVMWareClient(ctx, serverSpecs.Client, *serverSpecs.Configuration)
	if loginError != nil {
		return new(emptypb.Empty), status.Error(codes.Internal, fmt.Sprintf("Failed to login: %v", loginError))
	}

	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, executorContainer.VMID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting VM by UUID, %v", err))
	}

	err = library.AwaitVMToolsToComeOnline(ctx, virtualMachine, serverSpecs.Configuration.Variables)
	if err != nil {
		return nil, err
	}

	mutex, err := serverSpecs.MutexPool.GetMutex(ctx, executorContainer.VMID)
	if err != nil {
		return new(emptypb.Empty), err
	}
	if err = mutex.Lock(ctx); err != nil {
		return new(emptypb.Empty), err
	}
	deleteErr := vmwareClient.DeleteUploadedFiles(ctx, &executorContainer)
	if err := mutex.Unlock(ctx); err != nil {
		return new(emptypb.Empty), err
	}
	if deleteErr != nil {
		return new(emptypb.Empty), deleteErr
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
	feature.RegisterFeatureServiceServer(grpcServer, &featurerServer{ServerSpecs: &serverSpecs})
	condition.RegisterConditionServiceServer(grpcServer, &conditionerServer{ServerSpecs: &serverSpecs})
	inject.RegisterInjectServiceServer(grpcServer, &injectServer{ServerSpecs: &serverSpecs})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerType{
		*capability.Capabilities_Feature.Enum(),
		*capability.Capabilities_Condition.Enum(),
		*capability.Capabilities_Inject.Enum(),
	})

	capability.RegisterCapabilityServer(grpcServer, &capabilityServer)

	log.Printf("Executor listening at %v", listeningAddress.Addr())
	log.Printf("Version: %v", library.Version)
	log.Printf("Max Connections: %v", mutexPool.Configuration.MaxConnections)

	if bindError := grpcServer.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	validator := library.NewValidator()
	validator.SetRequireVSphereConfiguration(true)
	validator.SetRequireExerciseRootPath(true)
	validator.SetRequireRedisConfiguration(true)
	configuration, configurationError := validator.GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}
	RealMain(&configuration)
}
