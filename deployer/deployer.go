package main

import (
	"context"
	"fmt"
	"log"

	sdl_parser "github.com/open-cyber-range/sdl-parser"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

type Node struct {
	ExerciseName string
	NodeName     string
	TemplateName string
}

type Deployment struct {
	Client        *govmomi.Client
	Node          Node
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
	fmt.Println(exerciseFolder)

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
	task, taskError := template.Clone(context.Background(), exersiceFolder, deployment.Node.NodeName, cloneSpesifcation)
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

func main() {
	log.SetPrefix("deployer: ")
	log.SetFlags(0)

	parsedSDL, sdlError := sdl_parser.ParseSDL(`
scenario:
  name: test-scenario
  description: some-description
  start: 2022-01-20T13:00:00Z
  end: 2022-01-20T23:00:00Z
  infrastructure:
    #win10:
    #  type: VM
    #  description: win-10-description
    #  template: windows10
    #  flavor:
    #    ram: 4gb
    #    cpu: 2  
    deb10:
      type: VM
      description: deb-10-description
      template: debian10
      flavor:
        ram: 2gb
        cpu: 1

`)

	if sdlError != nil {
		log.Fatal(sdlError)
	}

	fmt.Println(parsedSDL)
	ctx := context.Background()

	configuration, configurationError := getConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}

	client, clientError := configuration.createClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	node := Node{
		ExerciseName: "test-scenario	",
		NodeName:     "test-node",
		TemplateName: "debian10",
	}

	deployment := Deployment{
		Client:        client,
		Node:          node,
		Configuration: configuration,
	}

	deploymentError := deployment.run()
	if deploymentError != nil {
		log.Fatal(deploymentError)
	}
	fmt.Println("Create success")
}
