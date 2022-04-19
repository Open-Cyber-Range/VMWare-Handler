package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	common "github.com/open-cyber-range/vmware-node-deployer/grpc/common"
	node "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
)

type Deployment struct {
	Client        *govmomi.Client
	Node          *node.Node
	Configuration *Configuration
}

func CreateFinder(client *govmomi.Client) *find.Finder {
	finder := find.NewFinder(client.Client, true)
	ctx := context.Background()
	datacenter, datacenterError := finder.DefaultDatacenter(ctx)
	if datacenterError != nil {
		log.Fatal(datacenterError)
	}
	finder.SetDatacenter(datacenter)
	return finder
}

func findTemplates(client *govmomi.Client, templatePath string) ([]*object.VirtualMachine, error) {
	finder := CreateFinder(client)
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
		if templateCandidate.Name() == deployment.Node.TemplateName {
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
	folderPath := deployment.Configuration.ExerciseRootPath + deployment.Node.ExerciseName

	existingFolder, _ := finder.Folder(ctx, folderPath)
	if existingFolder != nil {
		return existingFolder, nil
	}

	baseFolder, err := finder.Folder(ctx, deployment.Configuration.ExerciseRootPath)
	if err != nil {
		return
	}

	exerciseFolder, err := baseFolder.CreateFolder(ctx, deployment.Node.ExerciseName)
	if err != nil {
		return
	}

	return exerciseFolder, nil
}

func (deployment *Deployment) getResoucePool() (*object.ResourcePool, error) {
	ctx := context.Background()
	finder := find.NewFinder(deployment.Client.Client, true)
	datacenter, datacenterError := finder.DefaultDatacenter(ctx)
	if datacenterError != nil {
		return nil, fmt.Errorf("default datacenter not found")
	}
	finder.SetDatacenter(datacenter)
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
	task, err := template.Clone(context.Background(), exersiceFolder, deployment.Node.Name, cloneSpesifcation)
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

func (deployment *Deployment) delete() (err error) {
	finder := CreateFinder(deployment.Client)
	nodePath := deployment.Configuration.ExerciseRootPath + "/" + deployment.Node.ExerciseName + "/" + deployment.Node.Name
	virtualMachine, err := finder.VirtualMachine(context.Background(), nodePath)
	if err != nil {
		return
	}

	ctx := context.Background()

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

func (server *nodeServer) Create(ctx context.Context, node *node.Node) (*common.SimpleResponse, error) {
	deployment := Deployment{
		Client:        server.Client,
		Configuration: server.Configuration,
		Node:          node,
	}
	log.Printf("received node for deployement: %v in exercise: %v\n", node.Name, node.ExerciseName)

	deploymentError := deployment.create()
	if deploymentError != nil {
		return &common.SimpleResponse{Message: fmt.Sprintf("Node deployment failed due to: %v", deploymentError), Status: common.SimpleResponse_ERROR}, nil
	}
	log.Printf("deployed: %v", node.GetName())
	return &common.SimpleResponse{Message: "Deployed node: " + node.GetName(), Status: common.SimpleResponse_OK}, nil
}

func (server *nodeServer) Delete(ctx context.Context, Identifier *common.Identifier) (*common.SimpleResponse, error) {
	splitIdentifier := strings.Split(Identifier.GetValue(), "/")
	node := node.Node{
		Name:         splitIdentifier[1],
		ExerciseName: splitIdentifier[0],
	}
	deployment := Deployment{
		Client:        server.Client,
		Configuration: server.Configuration,
		Node:          &node,
	}
	log.Printf("Received node for deleting: %v in exercise: %v\n", node.Name, node.ExerciseName)

	deploymentError := deployment.delete()
	if deploymentError != nil {
		log.Printf("failed to delete node: %v\n", deploymentError)
		return &common.SimpleResponse{Message: fmt.Sprintf("Node deployment failed due to: %v", deploymentError), Status: common.SimpleResponse_ERROR}, nil
	}
	log.Printf("deleted: %v\n", node.GetName())
	return &common.SimpleResponse{Message: "Deployed node: " + node.GetName(), Status: common.SimpleResponse_OK}, nil
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
