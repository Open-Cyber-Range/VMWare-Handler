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
	Mutex         *library.Mutex
}

type vmWareTarget struct {
	VmID          string
	VmAccount     *library.Account
	PackageSource *common.Source
}

func (server *conditionerServer) Create(ctx context.Context, conditionDeployment *condition.Condition) (*common.Identifier, error) {
	var isSourceDeployment bool

	if conditionDeployment.Source != nil && conditionDeployment.Command != "" {
		return nil, status.Error(codes.Internal, fmt.Sprintln("Conflicting deployment info - A Condition deployment cannot have both `Command` and `Source`"))
	} else if conditionDeployment.Source != nil && conditionDeployment.Command == "" {
		isSourceDeployment = true
	}

	vmAuthentication := &types.NamePasswordAuthentication{
		Username: conditionDeployment.GetAccount().GetUsername(),
		Password: conditionDeployment.GetAccount().GetPassword(),
	}
	vmwareClient := library.NewVMWareClient(server.ServerSpecs.Client, server.ServerSpecs.Configuration.TemplateFolderPath)

	guestManager, err := vmwareClient.CreateGuestManagers(ctx, conditionDeployment.GetVirtualMachineId(), vmAuthentication)
	if err != nil {
		return nil, err
	}
	if _, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine); err != nil {
		return nil, err
	}

	var commandAction string
	var interval int32
	var assetFilePaths []string

	if isSourceDeployment {
		if err = server.ServerSpecs.Mutex.Lock(); err != nil {
			return nil, err
		}
		packagePath, packageMetadata, err := library.GetPackageMetadata(
			conditionDeployment.GetSource().GetName(),
			conditionDeployment.GetSource().GetVersion(),
		)
		if err != nil {
			return nil, err
		}

		commandAction = packageMetadata.Condition.Action
		interval = int32(packageMetadata.Condition.Interval)

		assetFilePaths, err = guestManager.CopyAssetsToVM(ctx, packageMetadata.Condition.Assets, packagePath)
		if err != nil {
			return nil, err
		}
		if err = library.CleanupTempPackage(packagePath); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error during temp Condition package cleanup (%v)", err))
		}
		if err := server.ServerSpecs.Mutex.Unlock(); err != nil {
			return nil, err
		}
	} else {
		commandAction = conditionDeployment.GetCommand()
		interval = conditionDeployment.GetInterval()
		assetFilePaths = append(assetFilePaths, commandAction)
	}

	server.ServerSpecs.Storage.Container = library.ExecutorContainer{
		VMID:      conditionDeployment.GetVirtualMachineId(),
		Auth:      *guestManager.Auth,
		FilePaths: assetFilePaths,
		Command:   commandAction,
		Interval:  interval,
	}
	conditionId := uuid.New().String()
	if err = server.ServerSpecs.Storage.Create(ctx, conditionId); err != nil {
		return nil, err
	}

	return &common.Identifier{Value: conditionId}, nil
}

func (server *conditionerServer) Stream(identifier *common.Identifier, stream condition.ConditionService_StreamServer) error {
	ctx := context.Background()

	container, err := server.ServerSpecs.Storage.Get(context.Background(), identifier.GetValue())
	if err != nil {
		return err
	}

	vmwareClient := library.NewVMWareClient(server.ServerSpecs.Client, server.ServerSpecs.Configuration.TemplateFolderPath)
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, container.VMID, &container.Auth)
	if err != nil {
		return err
	}
	if _, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine); err != nil {
		return err
	}
	for {
		if err = server.ServerSpecs.Mutex.Lock(); err != nil {
			return err
		}
		commandReturnValue, err := guestManager.ExecutePackageAction(ctx, container.Command)
		if err != nil {
			return err
		}
		if err := server.ServerSpecs.Mutex.Unlock(); err != nil {
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
	return uninstallPackage(ctx, server.ServerSpecs, identifier)
}

func (server *featurerServer) Create(ctx context.Context, featureDeployment *feature.Feature) (*feature.FeatureResponse, error) {
	vmWareTarget := vmWareTarget{
		VmAccount: &library.Account{
			Name:     featureDeployment.GetAccount().GetUsername(),
			Password: featureDeployment.GetAccount().GetPassword(),
		},
		PackageSource: featureDeployment.GetSource(),
		VmID:          featureDeployment.GetVirtualMachineId(),
	}
	identifier, vmLog, err := installPackage(ctx, &vmWareTarget, server.ServerSpecs)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during Feature Create: %v", err))
	}

	return &feature.FeatureResponse{
		Identifier: identifier,
		VmLog:      vmLog,
	}, nil
}

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	return uninstallPackage(ctx, server.ServerSpecs, identifier)
}

func (server *injectServer) Create(ctx context.Context, injectDeployment *inject.Inject) (*common.Identifier, error) {

	vmWareTarget := vmWareTarget{
		VmID: injectDeployment.GetVirtualMachineId(),
		VmAccount: &library.Account{
			Name:     injectDeployment.GetAccount().GetUsername(),
			Password: injectDeployment.GetAccount().GetPassword(),
		},
		PackageSource: injectDeployment.GetSource(),
	}
	identifier, _, err := installPackage(ctx, &vmWareTarget, server.ServerSpecs)

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during Inject Create: %v", err))
	}
	return identifier, nil
}

func (server *injectServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	return uninstallPackage(ctx, server.ServerSpecs, identifier)
}

func installPackage(ctx context.Context, vmWareTarget *vmWareTarget, serverSpecs *serverSpecs) (*common.Identifier, string, error) {
	vmAuthentication := &types.NamePasswordAuthentication{
		Username: vmWareTarget.VmAccount.Name,
		Password: vmWareTarget.VmAccount.Password,
	}

	vmwareClient := library.NewVMWareClient(serverSpecs.Client, serverSpecs.Configuration.TemplateFolderPath)
	guestManager, err := vmwareClient.CreateGuestManagers(ctx, vmWareTarget.VmID, vmAuthentication)
	if err != nil {
		return nil, "", err
	}

	if err = serverSpecs.Mutex.Lock(); err != nil {
		return nil, "", err
	}
	packageMetadata, assetFilePaths, err := guestManager.UploadPackageContents(ctx, vmWareTarget.PackageSource)
	if err != nil {
		return nil, "", err
	}

	vmPackageUuid := uuid.New().String()
	serverSpecs.Storage.Container = library.ExecutorContainer{
		VMID:      vmWareTarget.VmID,
		Auth:      *guestManager.Auth,
		FilePaths: assetFilePaths,
	}

	if err = serverSpecs.Storage.Create(ctx, vmPackageUuid); err != nil {
		return nil, "", err
	}

	packageAction := packageMetadata.GetAction()
	var vmLog string
	if packageAction != "" {
		vmLog, err = guestManager.ExecutePackageAction(ctx, packageAction)
		if err != nil {
			return nil, "", err
		}
	}
	if err := serverSpecs.Mutex.Unlock(); err != nil {
		return nil, "", err
	}

	return &common.Identifier{
			Value: fmt.Sprintf("%v", vmPackageUuid),
		},
		vmLog,
		nil
}

func uninstallPackage(ctx context.Context, serverSpecs *serverSpecs, identifier *common.Identifier) (*emptypb.Empty, error) {
	executorContainer, err := serverSpecs.Storage.Get(ctx, identifier.GetValue())
	if err != nil {
		return nil, err
	}

	vmwareClient := library.NewVMWareClient(serverSpecs.Client, serverSpecs.Configuration.TemplateFolderPath)

	if err = serverSpecs.Mutex.Lock(); err != nil {
		return new(emptypb.Empty), err
	}
	if err = vmwareClient.DeleteUploadedFiles(ctx, &executorContainer); err != nil {
		return nil, err
	}
	if err := serverSpecs.Mutex.Unlock(); err != nil {
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

	serverSpecs := serverSpecs{
		Client:        govmomiClient,
		Configuration: configuration,
		Storage:       &storage,
		Mutex:         &library.Mutex{Mutex: mutex},
	}

	feature.RegisterFeatureServiceServer(grpcServer, &featurerServer{ServerSpecs: &serverSpecs})
	condition.RegisterConditionServiceServer(grpcServer, &conditionerServer{ServerSpecs: &serverSpecs})
	inject.RegisterInjectServiceServer(grpcServer, &injectServer{ServerSpecs: &serverSpecs})

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
