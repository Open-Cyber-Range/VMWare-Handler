package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"path"

	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
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
	Client        *govmomi.Client
	Node          *node.Node
	Configuration *Configuration
	Parameters    *node.DeploymentParameters
}

func createFinderAndDatacenter(client *govmomi.Client) (*find.Finder, *object.Datacenter, error) {
	finder := find.NewFinder(client.Client, true)
	ctx := context.Background()
	datacenter, datacenterError := finder.DefaultDatacenter(ctx)
	if datacenterError != nil {
		return nil, nil, datacenterError
	}
	finder.SetDatacenter(datacenter)
	return finder, datacenter, nil
}

func findTemplates(client *govmomi.Client, templatePath string) ([]*object.VirtualMachine, error) {
	finder, _, datacenterError := createFinderAndDatacenter(client)
	if datacenterError != nil {
		return nil, datacenterError
	}
	ctx := context.Background()
	return finder.VirtualMachineList(ctx, templatePath)
}

func (deployment *Deployment) getTemplate() (*object.VirtualMachine, error) {
	templates, templatesError := findTemplates(deployment.Client, deployment.Configuration.TemplateFolderPath)
	if templatesError != nil {
		return nil, templatesError
	}

	var template *object.VirtualMachine
	for _, templateCandidate := range templates {
		if templateCandidate.Name() == deployment.Parameters.TemplateName {
			template = templateCandidate
		}
	}

	if template == nil {
		return nil, fmt.Errorf("template not found")
	}

	return template, nil
}

func (deployment *Deployment) createOrFindExerciseFolder() (_ *object.Folder, err error) {
	finder := find.NewFinder(deployment.Client.Client, true)
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
		return
	}

	return exerciseFolder, nil
}

func (deployment *Deployment) getResoucePool() (*object.ResourcePool, error) {
	ctx := context.Background()
	finder, _, datacenterError := createFinderAndDatacenter(deployment.Client)
	if datacenterError != nil {
		return nil, datacenterError
	}
	resourcePool, poolError := finder.ResourcePool(ctx, deployment.Configuration.ResourcePoolPath)
	if poolError != nil {
		return nil, poolError
	}

	return resourcePool, nil
}

func (deployment *Deployment) create() (err error) {
	template, err := deployment.getTemplate()
	if err != nil {
		return
	}
	exersiceFolder, err := deployment.createOrFindExerciseFolder()
	if err != nil {
		return
	}
	resourcePool, err := deployment.getResoucePool()
	if err != nil {
		return
	}
	resourcePoolReference := resourcePool.Reference()
	cloneSpesifcation := types.VirtualMachineCloneSpec{
		PowerOn: true,
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

func waitForTaskSuccess(task *object.Task) error {
	ctx := context.Background()
	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return err
	}

	if info.State == types.TaskInfoStateSuccess {
		return nil
	}

	return fmt.Errorf("failed to perform task: %v", task.Name())
}

func (deployment *Deployment) getVirtualMachineByUUID(ctx context.Context, uuid string) (virtualMachine *object.VirtualMachine, virtualMachineRefError error) {
	_, datacenter, datacenterError := createFinderAndDatacenter(deployment.Client)
	if datacenterError != nil {
		return nil, datacenterError
	}
	searchIndex := object.NewSearchIndex(deployment.Client.Client)
	virtualMachineRef, virtualMachineRefError := searchIndex.FindByUuid(ctx, datacenter, uuid, true, nil)
	if virtualMachineRefError != nil {
		return
	}
	virtualMachine = object.NewVirtualMachine(deployment.Client.Client, virtualMachineRef.Reference())
	return
}

func (deployment *Deployment) delete(uuid string) (err error) {
	ctx := context.Background()
	virtualMachine, _ := deployment.getVirtualMachineByUUID(ctx, uuid)

	powerOffTask, err := virtualMachine.PowerOff(ctx)
	if err != nil {
		return
	}
	err = waitForTaskSuccess(powerOffTask)
	if err != nil {
		return
	}
	destroyTask, err := virtualMachine.Destroy(ctx)
	if err != nil {
		return
	}
	err = waitForTaskSuccess(destroyTask)
	return
}

type nodeServer struct {
	node.UnimplementedNodeServiceServer
	Client        *govmomi.Client
	Configuration *Configuration
}

func (server *nodeServer) Create(ctx context.Context, nodeDeployment *node.NodeDeployment) (*node.NodeIdentifier, error) {

	if nodeDeployment.GetNode().GetIdentifier().GetNodeType() == *node.NodeType_switch.Enum() {
		nodeIdentifier, err := CreateVirtualSwitch(ctx, nodeDeployment)
		if err != nil {
			return nil, err
		}
		return nodeIdentifier, nil
	}

	deployment := Deployment{
		Client:        server.Client,
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
	finder, _, datacenterError := createFinderAndDatacenter(deployment.Client)
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

	status.New(codes.OK, "Node creation successful")
	return &node.NodeIdentifier{
		Identifier: &common.Identifier{
			Value: virtualMachine.UUID(ctx),
		},
		NodeType: node.NodeType_vm,
	}, nil
}

func (server *nodeServer) Delete(ctx context.Context, nodeIdentifier *node.NodeIdentifier) (*emptypb.Empty, error) {

	uuid := nodeIdentifier.Identifier.GetValue()
	deployment := Deployment{
		Client:        server.Client,
		Configuration: server.Configuration,
	}

	virtualMachine, _ := deployment.getVirtualMachineByUUID(ctx, uuid)
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

	deploymentError := deployment.delete(uuid)
	if deploymentError != nil {
		log.Printf("failed to delete node: %v\n", deploymentError)
		status.New(codes.Internal, fmt.Sprintf("Delete: Error during deletion (%v)", deploymentError))
		return nil, deploymentError
	}
	log.Printf("deleted: %v\n", parameters.GetName())
	status.New(codes.OK, fmt.Sprintf("Node %v deleted", parameters.GetName()))
	return new(emptypb.Empty), nil
}

func RealMain(configuration *Configuration) {
	ctx := context.Background()
	client, clientError := configuration.createClient(ctx)
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
	log.Printf("server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("failed to serve: %v", bindError)
	}
}

func main() {
	log.SetPrefix("deployer: ")
	log.SetFlags(0)

	configuration, configurationError := getConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}

	RealMain(configuration)
}
