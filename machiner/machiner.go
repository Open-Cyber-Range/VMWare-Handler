package main

import (
	"context"
	"fmt"
	"net"
	"path"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	common "github.com/open-cyber-range/vmware-handler/grpc/common"
	virtual_machine "github.com/open-cyber-range/vmware-handler/grpc/virtual-machine"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Deployment struct {
	Client         *library.VMWareClient
	VirtualMachine *virtual_machine.VirtualMachine
	Configuration  *library.Configuration
	MetaInfo       *common.MetaInfo
}

func (deployment *Deployment) createOrFindDeploymentFolder(call_count int) (_ *object.Folder, err error) {
	finder, _, err := deployment.Client.CreateFinderAndDatacenter()
	if err != nil {
		return
	}
	ctx := context.Background()
	baseFolder, err := finder.Folder(ctx, deployment.Configuration.ExerciseRootPath)
	if err != nil {
		return
	}

	exerciseFolderPath := path.Join(deployment.Configuration.ExerciseRootPath, deployment.MetaInfo.ExerciseName)
	exerciseFolder, err := finder.Folder(ctx, exerciseFolderPath)
	if exerciseFolder == nil && err != nil {
		exerciseFolder, err = baseFolder.CreateFolder(ctx, deployment.MetaInfo.ExerciseName)
	}

	if err != nil {
		if call_count < 3 {
			return deployment.createOrFindDeploymentFolder(call_count + 1)
		}
		return
	}

	deploymentFolderPath := path.Join(exerciseFolderPath, deployment.MetaInfo.DeploymentName)
	deploymentFolder, err := finder.Folder(ctx, deploymentFolderPath)
	if deploymentFolder == nil && err != nil {
		deploymentFolder, err = exerciseFolder.CreateFolder(ctx, deployment.MetaInfo.DeploymentName)
	}

	if err != nil {
		if call_count < 3 {
			return deployment.createOrFindDeploymentFolder(call_count + 1)
		}
		return
	}

	return deploymentFolder, nil
}

func (deployment *Deployment) create() (err error) {
	ctx := context.Background()
	template, err := deployment.Client.GetVirtualMachineByUUID(ctx, deployment.VirtualMachine.TemplateId)
	if template == nil {
		return fmt.Errorf("template not found, uuid: %v", deployment.VirtualMachine.TemplateId)
	}
	if err != nil {
		return
	}
	deploymentFolder, err := deployment.createOrFindDeploymentFolder(0)
	if err != nil {
		return
	}
	resourcePool, err := deployment.Client.GetResourcePool(deployment.Configuration.ResourcePoolPath)
	if err != nil {
		return
	}
	resourcePoolReference := resourcePool.Reference()

	vmConfiguration := types.VirtualMachineConfigSpec{}
	if deployment.VirtualMachine.Configuration != nil {
		ramBytesAsMegabytes := (int64(deployment.VirtualMachine.Configuration.GetRam()) >> 20)
		vmConfiguration.NumCPUs = int32(deployment.VirtualMachine.Configuration.GetCpu())
		vmConfiguration.MemoryMB = ramBytesAsMegabytes
	}

	cloneSpesifcation := types.VirtualMachineCloneSpec{
		PowerOn: true,
		Config:  &vmConfiguration,
		Location: types.VirtualMachineRelocateSpec{
			Pool: &resourcePoolReference,
		},
	}
	task, err := template.Clone(context.Background(), deploymentFolder, deployment.VirtualMachine.Name, cloneSpesifcation)
	if err != nil {
		return
	}

	info, err := task.WaitForResult(context.Background())
	if err != nil {
		return
	}

	if info.State == types.TaskInfoStateSuccess {
		return nil
	}

	return fmt.Errorf("failed to clone template")
}

func createLinks(ctx context.Context, linkNames []string, finder *find.Finder) (object.VirtualDeviceList, error) {
	var links object.VirtualDeviceList

	for _, linkName := range linkNames {
		network, networkFetchError := finder.Network(context.Background(), linkName)
		if networkFetchError != nil {
			return nil, fmt.Errorf("failed to fetch network (%v)", networkFetchError)
		}

		ethernetBacking, ethernetBackingError := network.EthernetCardBackingInfo(ctx)
		if ethernetBackingError != nil {
			return nil, fmt.Errorf("failed to create Ethernet card backing info (%v)", ethernetBackingError)
		}

		ethernetCard, ethernetCardError := object.EthernetCardTypes().CreateEthernetCard("vmxnet3", ethernetBacking)
		if ethernetCardError != nil {
			return nil, fmt.Errorf("failed to create Ethernet card (%v)", ethernetCardError)
		}

		links = append(links, ethernetCard)
	}

	return links, nil
}

func addLinks(ctx context.Context, links object.VirtualDeviceList, virtualMachine object.VirtualMachine) error {
	for _, link := range links {
		err := virtualMachine.AddDevice(ctx, link)
		if err != nil {
			return err
		}
	}

	return nil
}

type virtualMachineServer struct {
	virtual_machine.UnimplementedVirtualMachineServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
}

func (server *virtualMachineServer) Create(ctx context.Context, virtualMachineDeployment *virtual_machine.DeployVirtualMachine) (*common.Identifier, error) {
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	deployment := Deployment{
		Client:         &vmwareClient,
		Configuration:  server.Configuration,
		VirtualMachine: virtualMachineDeployment.VirtualMachine,
		MetaInfo:       virtualMachineDeployment.MetaInfo,
	}
	deployment.VirtualMachine.Name = library.SanitizeToCompatibleName((deployment.VirtualMachine.Name))
	log.Infof("Received node: %v for deployment in exercise: %v, deployment: %v", deployment.VirtualMachine.Name, deployment.MetaInfo.ExerciseName, deployment.MetaInfo.DeploymentName)
	deploymentError := deployment.create()
	if deploymentError != nil {
		log.Errorf("Deployment creation error (%v)", deploymentError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Deployment creation error (%v)", deploymentError))
	}
	finder, _, datacenterError := vmwareClient.CreateFinderAndDatacenter()
	if datacenterError != nil {
		log.Errorf("Datacenter creation error (%v)", datacenterError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Datacenter creation error (%v)", datacenterError))
	}
	nodePath := path.Join(deployment.Configuration.ExerciseRootPath, deployment.MetaInfo.ExerciseName, deployment.MetaInfo.DeploymentName, deployment.VirtualMachine.Name)
	virtualMachine, virtualMachineError := finder.VirtualMachine(context.Background(), nodePath)
	if virtualMachineError != nil {
		log.Errorf("Node creation error (%v)", virtualMachineError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Node creation error (%v)", virtualMachineError))
	}
	log.Infof("Deployed: %v", deployment.VirtualMachine.GetName())

	links, linkCreationError := createLinks(ctx, deployment.VirtualMachine.Links, finder)
	if linkCreationError != nil {
		log.Errorf("Link creation error (%v)", linkCreationError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Link creation error (%v)", linkCreationError))
	}

	linkAddingError := addLinks(ctx, links, *virtualMachine)
	if linkAddingError != nil {
		log.Errorf("Adding links to node error (%v)", linkAddingError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Adding links to node error (%v)", linkAddingError))
	}

	log.Infof("Node: %v deployed  in exercise: %v, deployment: %v", deployment.VirtualMachine.Name, deployment.MetaInfo.ExerciseName, deployment.MetaInfo.DeploymentName)
	return &common.Identifier{
		Value: virtualMachine.UUID(ctx),
	}, nil
}

func (server *virtualMachineServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	uuid := identifier.GetValue()
	deployment := Deployment{
		Client:        &vmwareClient,
		Configuration: server.Configuration,
	}

	virtualMachine, _ := deployment.Client.GetVirtualMachineByUUID(ctx, uuid)
	virtualMachineName, virtualMachineNameError := virtualMachine.ObjectName(ctx)
	if virtualMachineNameError != nil {
		log.Errorf("Node name retrieval error (%v)", virtualMachineNameError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Node name retrieval error (%v)", virtualMachineNameError))
	}

	log.Infof("Received node: %v for deleting with UUID: %v", virtualMachineName, uuid)

	deploymentError := deployment.Client.DeleteVirtualMachineByUUID(uuid)
	if deploymentError != nil {
		log.Errorf("Error during node deletion (%v)", deploymentError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during node deletion (%v)", deploymentError))
	}
	log.Infof("Deleted node: %v", virtualMachineName)
	return new(emptypb.Empty), nil
}

func RealMain(configuration *library.Configuration) {
	ctx := context.Background()
	client, clientError := configuration.CreateClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	listeningAddress, addressError := net.Listen("tcp", configuration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
	}

	server := grpc.NewServer()
	virtual_machine.RegisterVirtualMachineServiceServer(server, &virtualMachineServer{
		Client:        client,
		Configuration: configuration,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_VirtualMachine.Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("Machiner listening at %v", listeningAddress.Addr())
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
