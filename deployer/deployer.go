package main

import (
	"context"
	"fmt"
	"log"
	"net"

	// pb "github.com/open-cyber-range/vmware-node-deployer/grpc/node"
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
	Node          node.Node
	Configuration *Configuration
}

func findTemplates(client *govmomi.Client, templatePath string) ([]*object.VirtualMachine, error) {
	finder := find.NewFinder(client.Client, true)

	ctx := context.Background()
	datacenter, datacenterError := finder.DefaultDatacenter(ctx)
	if datacenterError != nil {
		log.Fatal(datacenterError)
	}
	finder.SetDatacenter(datacenter)

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

func (deployment *Deployment) createOrFindExerciseFolder() (*object.Folder, error) {
	finder := find.NewFinder(deployment.Client.Client, true)
	ctx := context.Background()
	folderPath := deployment.Configuration.ExerciseRootPath + deployment.Node.ExerciseName

	existingFolder, _ := finder.Folder(ctx, folderPath)
	if existingFolder != nil {
		return existingFolder, nil
	}

	baseFolder, baseFolderError := finder.Folder(ctx, deployment.Configuration.ExerciseRootPath)
	if baseFolderError != nil {
		return nil, baseFolderError
	}

	exerciseFolder, errorCreatingFolder := baseFolder.CreateFolder(ctx, deployment.Node.ExerciseName)
	if errorCreatingFolder != nil {
		return nil, errorCreatingFolder
	}

	return exerciseFolder, nil
}

func (deployment *Deployment) getResoucePool() (*object.ResourcePool, error) {
	ctx := context.Background()
	finder := find.NewFinder(deployment.Client.Client, true)
	datacenter, datacenterError := finder.DefaultDatacenter(ctx)
	if datacenterError != nil {
		log.Fatal(datacenterError)
	}
	finder.SetDatacenter(datacenter)
	resourcePool, poolError := finder.ResourcePool(ctx, deployment.Configuration.ResourcePoolPath)
	if poolError != nil {
		return nil, poolError
	}

	return resourcePool, nil
}

func (deployment *Deployment) run() error {
	template, templateError := deployment.getTemplate()
	if templateError != nil {
		return templateError
	}
	exersiceFolder, folderError := deployment.createOrFindExerciseFolder()
	if folderError != nil {
		return folderError
	}
	resourcePool, poolError := deployment.getResoucePool()
	if poolError != nil {
		return poolError
	}
	resourcePoolReference := resourcePool.Reference()
	cloneSpesifcation := types.VirtualMachineCloneSpec{
		PowerOn: true,
		Location: types.VirtualMachineRelocateSpec{
			Pool: &resourcePoolReference,
		},
	}
	task, taskError := template.Clone(context.Background(), exersiceFolder, deployment.Node.Name, cloneSpesifcation)
	if taskError != nil {
		return taskError
	}

	info, infoError := task.WaitForResult(context.Background())
	if infoError != nil {
		return infoError
	}

	if info.State == types.TaskInfoStateSuccess {
		return nil
	}

	return fmt.Errorf("failed to clone template")
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
	}
	log.Printf("Received node for deployement: %v in exercise: %v", node.Name, node.ExerciseName)

	deploymentError := deployment.run()
	if deploymentError != nil {
		return &common.SimpleResponse{Message: fmt.Sprintf("Node deployment failed to: %v", deploymentError), Status: common.SimpleResponse_ERROR}, nil
	}
	log.Printf("Deployed: %v", node.GetName())
	return &common.SimpleResponse{Message: "Deployed node: " + node.GetName(), Status: common.SimpleResponse_OK}, nil
}

func main() {
	log.SetPrefix("deployer: ")
	log.SetFlags(0)
	ctx := context.Background()

	configuration, configurationError := getConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}

	client, clientError := configuration.createClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	listeningAddress, addressError := net.Listen("tcp", configuration.ServerPath)
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

// func main() {
// 	log.SetPrefix("deployer: ")
// 	log.SetFlags(0)

// 	ctx := context.Background()

// 	configuration, configurationError := getConfiguration()
// 	if configurationError != nil {
// 		log.Fatal(configurationError)
// 	}

// 	client, clientError := configuration.createClient(ctx)
// 	if clientError != nil {
// 		log.Fatal(clientError)
// 	}

// 	node := Node{
// 		ExerciseName: "test-scenario	",
// 		NodeName:     "test-node",
// 		TemplateName: "debian10",
// 	}

// 	deployment := Deployment{
// 		Client:        client,
// 		Node:          node,
// 		Configuration: configuration,
// 	}

// 	deploymentError := deployment.run()
// 	if deploymentError != nil {
// 		log.Fatal(deploymentError)
// 	}
// }
