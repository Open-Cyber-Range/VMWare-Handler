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
	Client           *govmomi.Client
	Configuration    *library.Configuration
	Storage          *library.Storage[library.ExecutorContainer]
	MutexDistributor *library.MutexDistributor
}

type vmWareTarget struct {
	VmID          string
	VmAccount     *library.Account
	PackageSource *common.Source
	Environment   []string
}

func (server *conditionerServer) deployConditionToVm(ctx context.Context, conditionDeployment *condition.Condition, guestManager library.GuestManager) (commandAction string, interval int32, assetFilePaths []string, err error) {
	var isSourceDeployment bool

	if conditionDeployment.Source != nil && conditionDeployment.Command != "" {
		log.Errorf("Conflicting deployment info - A Condition deployment cannot have both `Command` and `Source`")
		return "", 0, nil, fmt.Errorf("conflicting deployment info - A Condition deployment cannot have both `Command` and `Source`")
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
			return "", 0, nil, err
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
		log.Errorf("Error creating vmwareClient %v", loginError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating vmwareClient (%v)", loginError))
	}
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, conditionDeployment.GetVirtualMachineId(), vmAuthentication)
	if err != nil {
		log.Errorf("Error creating guest managers: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating guest managers: %v", err))
	}
	if err = library.AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, server.ServerSpecs.Configuration.Variables); err != nil {
		log.Errorf("Error awaiting VMTools to come online: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error awaiting VMTools to come online: %v", err))
	}

	mutexOptions := library.NewMutexOptions(conditionDeployment.GetVirtualMachineId(), conditionDeployment.Name, int(conditionDeployment.Interval))
	mutex, err := server.ServerSpecs.MutexDistributor.GetMutex(ctx, mutexOptions)
	if err != nil {
		log.Errorf("Error getting mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting mutex: %v", err))
	}
	if err = mutex.Lock(ctx); err != nil {
		log.Errorf("Error locking mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error locking mutex: %v", err))
	}
	commandAction, interval, assetFilePaths, createErr := server.deployConditionToVm(ctx, conditionDeployment, *guestManager)
	if err := mutex.Unlock(); err != nil {
		log.Errorf("Error unlocking mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error unlocking mutex: %v", err))
	}
	if createErr != nil {
		log.Errorf("Error deploying condition to vm: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error deploying condition to vm: %v", createErr))
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
		log.Errorf("Error creating package metadata storage: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating package metadata storage: %v", err))
	}

	return &common.Identifier{Value: conditionId}, nil
}

func (server *conditionerServer) Stream(identifier *common.Identifier, stream condition.ConditionService_StreamServer) error {
	ctx := context.Background()
	container, err := server.ServerSpecs.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		log.Errorf("Error getting package metadata from storage: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("Error getting package metadata from storage: %v", err))
	}

	vmwareClient, loginError := library.NewVMWareClient(ctx, server.ServerSpecs.Client, *server.ServerSpecs.Configuration)
	if loginError != nil {
		log.Errorf("Error logging into VMWare client: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("Error logging into VMWare client: %v", loginError))
	}
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, container.VMID, &container.Auth)
	if err != nil {
		log.Errorf("Error creating guest managers: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("Error creating guest managers: %v", err))
	}
	mutexOptions := library.NewMutexOptions(container.VMID, identifier.GetValue(), int(container.Interval))

	for {
		if err = library.AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, server.ServerSpecs.Configuration.Variables); err != nil {
			log.Errorf("Error waiting for VM tools to come online: %v", err)
			return status.Error(codes.Internal, fmt.Sprintf("Error waiting for VM tools to come online: %v", err))
		}
		mutex, err := server.ServerSpecs.MutexDistributor.GetMutex(ctx, mutexOptions)
		if err != nil {
			log.Errorf("Error getting mutex: %v", err)
			return status.Error(codes.Internal, fmt.Sprintf("Error getting mutex: %v", err))
		}
		if err = mutex.Lock(ctx); err != nil {
			log.Errorf("Error locking mutex: %v", err)
			return status.Error(codes.Internal, fmt.Sprintf("Error locking mutex: %v", err))
		}

		stdout, _, executeErr := guestManager.ExecutePackageAction(ctx, container.Command, container.Environment)
		if err := mutex.Unlock(); err != nil {
			log.Errorf("Error unlocking mutex: %v", err)
			return status.Error(codes.Internal, fmt.Sprintf("Error unlocking mutex: %v", err))
		}
		if executeErr != nil {
			log.Errorf("Error executing condition package action %v", err)
			return status.Error(codes.Internal, fmt.Sprintf("Error executing condition package action: %v", executeErr))
		}
		conditionValue, err := strconv.ParseFloat(stdout, 32)
		if err != nil {
			log.Errorf("Error parsing condition return value %v. Error: %v", stdout, err)
			return status.Error(codes.Internal, fmt.Sprintf("Error parsing condition return value %v. Error: %v", stdout, err))
		}
		log.Tracef("Condition ID '%v' returned '%v'", identifier.GetValue(), conditionValue)

		message := &condition.ConditionStreamResponse{
			Response:           identifier.GetValue(),
			CommandReturnValue: float32(conditionValue),
		}

		if err = stream.Send(message); err != nil {
			log.Errorf("Error sending Condition stream: %v", err)
			return status.Error(codes.Internal, fmt.Sprintf("Error sending Condition stream: %v", err))
		}
		time.Sleep(time.Second * time.Duration(container.Interval))
	}
}

func (server *conditionerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	empty, err := uninstallPackage(ctx, server.ServerSpecs, identifier)
	if err != nil {
		log.Errorf("Error deleting condition package: %v", err)
		return empty, status.Error(codes.Internal, fmt.Sprintf("Error deleting condition: %v", err))
	}

	return empty, nil
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
	mutexOptions := library.NewMutexOptions(featureDeployment.GetVirtualMachineId(), featureDeployment.GetName())

	mutex, err := server.ServerSpecs.MutexDistributor.GetMutex(ctx, mutexOptions)
	if err != nil {
		log.Errorf("Error getting mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting mutex: %v", err))
	}
	if err = mutex.Lock(ctx); err != nil {
		log.Errorf("Error locking mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error locking mutex: %v", err))
	}
	executorResponse, installErr := installPackage(ctx, &vmWareTarget, server.ServerSpecs)
	if err := mutex.Unlock(); err != nil {
		log.Errorf("Error unlocking mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error unlocking mutex: %v", err))
	}
	if installErr != nil {
		log.Errorf("Error installing feature package on VM: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error installing feature package on VM: %v", installErr))
	}

	return executorResponse, nil
}

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	empty, err := uninstallPackage(ctx, server.ServerSpecs, identifier)
	if err != nil {
		log.Errorf("Error deleting feature: %v", err)
		return empty, status.Error(codes.Internal, fmt.Sprintf("Error deleting feature: %v", err))
	}

	return empty, nil
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
	mutexOptions := library.NewMutexOptions(injectDeployment.VirtualMachineId, injectDeployment.Name)

	mutex, err := server.ServerSpecs.MutexDistributor.GetMutex(ctx, mutexOptions)
	if err != nil {
		log.Errorf("Error getting mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting mutex: %v", err))
	}
	if err = mutex.Lock(ctx); err != nil {
		log.Errorf("Error locking mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error locking mutex: %v", err))
	}
	executorResponse, installErr := installPackage(ctx, &vmWareTarget, server.ServerSpecs)
	if err := mutex.Unlock(); err != nil {
		log.Errorf("Error unlocking mutex: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error unlocking mutex: %v", err))
	}
	if installErr != nil {
		log.Errorf("Error installing inject package on vm: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error installing inject package on vm: %v", installErr))
	}

	return executorResponse, nil
}

func (server *injectServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	empty, err := uninstallPackage(ctx, server.ServerSpecs, identifier)
	if err != nil {
		log.Errorf("Error deleting inject: %v", err)
		return empty, status.Error(codes.Internal, fmt.Sprintf("Error deleting inject: %v", err))
	}

	return empty, nil
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
	var stdout string
	var stderr string
	if packageAction != "" {
		stdout, stderr, err = guestManager.ExecutePackageAction(ctx, packageAction, vmWareTarget.Environment)
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
		Stdout: stdout,
		Stderr: stderr,
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

	mutexOptions := library.NewMutexOptions(executorContainer.VMID, identifier.GetValue(), int(executorContainer.Interval))

	mutex, err := serverSpecs.MutexDistributor.GetMutex(ctx, mutexOptions)
	if err != nil {
		return new(emptypb.Empty), err
	}
	if err = mutex.Lock(ctx); err != nil {
		return new(emptypb.Empty), err
	}
	deleteErr := vmwareClient.DeleteUploadedFiles(ctx, &executorContainer)
	if err := mutex.Unlock(); err != nil {
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

	mutexDistributor, err := library.NewMutexDistributor(ctx, configuration.Hostname, *redsync.New(redisPool), *redisClient, configuration.Variables.MaxConnections)
	if err != nil {
		log.Fatal(err)
	}

	serverSpecs := serverSpecs{
		Client:           govmomiClient,
		Configuration:    configuration,
		Storage:          &storage,
		MutexDistributor: &mutexDistributor,
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
	log.Printf("Max Connections: %v", mutexDistributor.MaxConnections)

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
