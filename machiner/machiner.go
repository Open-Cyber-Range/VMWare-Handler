package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"path"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	common "github.com/open-cyber-range/vmware-handler/grpc/common"
	node "github.com/open-cyber-range/vmware-handler/grpc/node"
	"github.com/open-cyber-range/vmware-handler/library"
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
	Client        *library.VMWareClient
	Node          *node.Node
	Configuration *library.Configuration
	Parameters    *node.DeploymentParameters
}

func (deployment *Deployment) createOrFindExerciseFolder(call_count int) (_ *object.Folder, err error) {
	finder, _, err := deployment.Client.CreateFinderAndDatacenter()
	if err != nil {
		return
	}
	ctx := context.Background()
	folderPath := path.Join(deployment.Configuration.ExerciseRootPath, deployment.Parameters.ExerciseName)

	existingFolder, _ := finder.Folder(ctx, folderPath)
	if existingFolder != nil {
		return existingFolder, nil
	}

	baseFolder, err := finder.Folder(ctx, deployment.Configuration.ExerciseRootPath)
	if err != nil {
		return
	}

	exerciseFolder, err := baseFolder.CreateFolder(ctx, deployment.Parameters.ExerciseName)
	if err != nil {
		if call_count < 3 {
			return deployment.createOrFindExerciseFolder(call_count + 1)
		}
		return
	}

	return exerciseFolder, nil
}

func (deployment *Deployment) create() (err error) {
	ctx := context.Background()
	template, err := deployment.Client.GetVirtualMachineByUUID(ctx, deployment.Parameters.TemplateId)
	if template == nil {
		return fmt.Errorf("template not found, uuid: %v", deployment.Parameters.TemplateId)
	}
	if err != nil {
		return
	}
	exersiceFolder, err := deployment.createOrFindExerciseFolder(0)
	if err != nil {
		return
	}
	resourcePool, err := deployment.Client.GetResourcePool(deployment.Configuration.ResourcePoolPath)
	if err != nil {
		return
	}
	resourcePoolReference := resourcePool.Reference()

	vmConfiguration := types.VirtualMachineConfigSpec{}
	if deployment.Node.Configuration != nil {
		ramBytesAsMegabytes := (int64(deployment.Node.Configuration.GetRam()) >> 20)
		vmConfiguration.NumCPUs = int32(deployment.Node.Configuration.GetCpu())
		vmConfiguration.MemoryMB = ramBytesAsMegabytes
	}

	cloneSpesifcation := types.VirtualMachineCloneSpec{
		PowerOn: true,
		Config:  &vmConfiguration,
		Location: types.VirtualMachineRelocateSpec{
			Pool: &resourcePoolReference,
		},
	}
	task, err := template.Clone(context.Background(), exersiceFolder, deployment.Parameters.Name, cloneSpesifcation)
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

func createLinks(ctx context.Context, nodeDeployment *node.NodeDeployment, finder *find.Finder) (object.VirtualDeviceList, error) {
	linkNames := nodeDeployment.Parameters.Links
	var links object.VirtualDeviceList

	for _, linkName := range linkNames {
		network, networkFetchError := finder.Network(context.Background(), linkName)
		if networkFetchError != nil {
			return nil, networkFetchError
		}

		ethernetBacking, ethernetBackingError := network.EthernetCardBackingInfo(ctx)
		if ethernetBackingError != nil {
			return nil, ethernetBackingError
		}

		ethernetCard, ethernetCardError := object.EthernetCardTypes().CreateEthernetCard("vmxnet3", ethernetBacking)
		if ethernetCardError != nil {
			return nil, ethernetCardError
		}

		links = append(links, ethernetCard)
	}

	return links, nil
}

func addLinks(ctx context.Context, links object.VirtualDeviceList, virtualMachine object.VirtualMachine) error {
	for _, link := range links {
		linkAddingError := virtualMachine.AddDevice(ctx, link)
		if linkAddingError != nil {
			return linkAddingError
		}
	}

	return nil
}

type nodeServer struct {
	node.UnimplementedNodeServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
}

func (server *nodeServer) Create(ctx context.Context, nodeDeployment *node.NodeDeployment) (*node.NodeIdentifier, error) {
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	deployment := Deployment{
		Client:        &vmwareClient,
		Configuration: server.Configuration,
		Node:          nodeDeployment.Node,
		Parameters:    nodeDeployment.Parameters,
	}
	log.Printf("received node for deployement: %v in exercise: %v\n", nodeDeployment.Parameters.Name, nodeDeployment.Parameters.ExerciseName)
	deploymentError := deployment.create()
	if deploymentError != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: deployment error (%v)", deploymentError))
		return nil, deploymentError
	}
	finder, _, datacenterError := vmwareClient.CreateFinderAndDatacenter()
	if datacenterError != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: datacenter error (%v)", datacenterError))
		return nil, datacenterError
	}
	nodePath := path.Join(deployment.Configuration.ExerciseRootPath, deployment.Parameters.ExerciseName, deployment.Parameters.Name)
	virtualMachine, virtualMachineErr := finder.VirtualMachine(context.Background(), nodePath)
	if virtualMachineErr != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: VM creation error (%v)", virtualMachineErr))
		return nil, virtualMachineErr
	}

	log.Printf("deployed: %v", nodeDeployment.Parameters.GetName())

	links, linkCreationError := createLinks(ctx, nodeDeployment, finder)
	if linkCreationError != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: link creation error (%v)", linkCreationError))
		return nil, linkCreationError
	}

	linkAddingError := addLinks(ctx, links, *virtualMachine)
	if linkAddingError != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: adding links to VM error (%v)", linkAddingError))
		return nil, linkAddingError
	}

	status.New(codes.OK, "Node creation successful")
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: virtualMachine.UUID(ctx),
		},
		NodeType: node.NodeType_vm,
	}, nil
}

func (server *nodeServer) Delete(ctx context.Context, nodeIdentifier *node.NodeIdentifier) (*emptypb.Empty, error) {
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	uuid := nodeIdentifier.Identifier.GetValue()
	deployment := Deployment{
		Client:        &vmwareClient,
		Configuration: server.Configuration,
	}

	virtualMachine, _ := deployment.Client.GetVirtualMachineByUUID(ctx, uuid)
	nodeName, nodeNameError := virtualMachine.ObjectName(ctx)
	if nodeNameError != nil {
		status.New(codes.Internal, fmt.Sprintf("Delete: node name retrieval error (%v)", nodeNameError))
		return nil, nodeNameError
	}
	parameters := node.DeploymentParameters{
		Name: nodeName,
	}
	deployment.Parameters = &parameters

	log.Printf("Received node for deleting: %v with UUID: %v\n", parameters.Name, uuid)

	deploymentError := deployment.Client.DeleteVirtualMachineByUUID(uuid)
	if deploymentError != nil {
		log.Printf("failed to delete node: %v\n", deploymentError)
		status.New(codes.Internal, fmt.Sprintf("Delete: Error during deletion (%v)", deploymentError))
		return nil, deploymentError
	}
	log.Printf("deleted: %v\n", parameters.GetName())
	status.New(codes.OK, fmt.Sprintf("Node %v deleted", parameters.GetName()))
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
		log.Fatalf("failed to listen: %v", addressError)
	}

	server := grpc.NewServer()
	node.RegisterNodeServiceServer(server, &nodeServer{
		Client:        client,
		Configuration: configuration,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_VirtualMachine.Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("failed to serve: %v", bindError)
	}
}

func main() {
	log.SetPrefix("machiner: ")
	log.SetFlags(0)
	configuration, configurationError := library.NewValidator().SetRequireExerciseRootPath(true).GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}

	RealMain(&configuration)
}
