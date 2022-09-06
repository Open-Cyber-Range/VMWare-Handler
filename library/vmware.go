package library

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

type VMWareClient struct {
	Client       *govmomi.Client
	templatePath string
}

func NewVMWareClient(client *govmomi.Client, templatePath string) (configuration VMWareClient) {
	return VMWareClient{
		Client:       client,
		templatePath: templatePath,
	}
}

func (client *VMWareClient) CreateFinderAndDatacenter() (finder *find.Finder, datacenter *object.Datacenter, err error) {
	finder = find.NewFinder(client.Client.Client, true)
	ctx := context.Background()
	datacenter, err = finder.DefaultDatacenter(ctx)
	if err != nil {
		return
	}
	finder.SetDatacenter(datacenter)
	return
}

func (client *VMWareClient) findTemplates() ([]*object.VirtualMachine, error) {
	finder, _, datacenterError := client.CreateFinderAndDatacenter()
	if datacenterError != nil {
		return nil, datacenterError
	}
	ctx := context.Background()
	return finder.VirtualMachineList(ctx, client.templatePath)
}

func (client *VMWareClient) DoesTemplateExist(name string) (value bool, err error) {
	templates, err := client.findTemplates()
	if err != nil {
		return
	}
	for _, template := range templates {
		if template.Name() == name {
			return true, nil
		}
	}
	return false, nil
}

func (client *VMWareClient) GetTemplateByName(name string) (virtualMachine *object.VirtualMachine, err error) {
	finder, _, datacenterError := client.CreateFinderAndDatacenter()
	if datacenterError != nil {
		return nil, datacenterError
	}
	ctx := context.Background()
	return finder.VirtualMachine(ctx, path.Join(path.Dir(client.templatePath), name))
}

func (client *VMWareClient) GetTemplateFolder() (*object.Folder, error) {
	finder, _, datacenterError := client.CreateFinderAndDatacenter()
	if datacenterError != nil {
		return nil, datacenterError
	}
	ctx := context.Background()
	templateDirectoryPath := path.Dir(client.templatePath)

	return finder.Folder(ctx, templateDirectoryPath)
}

func (client *VMWareClient) GetTemplate(templateName string) (*object.VirtualMachine, error) {
	templates, templatesError := client.findTemplates()
	if templatesError != nil {
		return nil, templatesError
	}

	var template *object.VirtualMachine
	for _, templateCandidate := range templates {
		if templateCandidate.Name() == templateName {
			template = templateCandidate
		}
	}

	if template == nil {
		return nil, fmt.Errorf("template not found")
	}

	return template, nil
}

func (client *VMWareClient) GetResourcePool(resourcePoolPath string) (*object.ResourcePool, error) {
	ctx := context.Background()
	finder, _, datacenterError := client.CreateFinderAndDatacenter()
	if datacenterError != nil {
		return nil, datacenterError
	}
	resourcePool, poolError := finder.ResourcePool(ctx, resourcePoolPath)
	if poolError != nil {
		return nil, poolError
	}

	return resourcePool, nil
}

func (client *VMWareClient) GetDatastore(datastorePath string) (datastore *object.Datastore, err error) {
	ctx := context.Background()
	finder, _, err := client.CreateFinderAndDatacenter()
	if err != nil {
		return
	}
	datastore, err = finder.Datastore(ctx, datastorePath)
	return
}

func (client *VMWareClient) GetVirtualMachineByUUID(ctx context.Context, uuid string) (virtualMachine *object.VirtualMachine, virtualMachineRefError error) {
	_, datacenter, datacenterError := client.CreateFinderAndDatacenter()
	if datacenterError != nil {
		return nil, datacenterError
	}
	searchIndex := object.NewSearchIndex(client.Client.Client)
	virtualMachineRef, virtualMachineRefError := searchIndex.FindByUuid(ctx, datacenter, uuid, true, nil)
	if virtualMachineRefError != nil {
		return
	}
	if virtualMachineRef == nil {
		return nil, fmt.Errorf("virtual machine not found")
	}
	virtualMachine = object.NewVirtualMachine(client.Client.Client, virtualMachineRef.Reference())
	return
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

func (client *VMWareClient) DeleteVirtualMachineByUUID(uuid string) (err error) {
	ctx := context.Background()
	virtualMachine, err := client.GetVirtualMachineByUUID(ctx, uuid)
	if err != nil {
		return
	}
	powerState, err := virtualMachine.PowerState(ctx)
	if err != nil {
		return
	}

	if powerState != types.VirtualMachinePowerStatePoweredOff {
		var powerOffTask *object.Task
		powerOffTask, err = virtualMachine.PowerOff(ctx)
		if err != nil {
			return
		}
		err = waitForTaskSuccess(powerOffTask)
		if err != nil {
			return
		}
	}

	destroyTask, err := virtualMachine.Destroy(ctx)
	if err != nil {
		return
	}
	err = waitForTaskSuccess(destroyTask)
	return
}

func (client *VMWareClient) findLinks(ctx context.Context, linkNames []string) (networkNames []string, err error) {
	finder, _, _ := client.CreateFinderAndDatacenter()
	for _, linkName := range linkNames {
		network, err := finder.Network(ctx, linkName)
		if err != nil {
			return nil, fmt.Errorf("failed to find network %v (%v)", linkName, err)
		}
		networkNames = append(networkNames, network.Reference().Value)
	}
	return networkNames, nil
}

func (client *VMWareClient) CheckVMLinks(ctx context.Context, vmNetworks []types.ManagedObjectReference, linkNames []string) error {
	if vmNetworks == nil {
		return fmt.Errorf("failed to retrieve VM network list")
	}

	var vmNetworkNames string
	for _, network := range vmNetworks {
		vmNetworkNames = vmNetworkNames + " " + network.Value
	}

	networkNames, err := client.findLinks(ctx, linkNames)
	if err != nil {
		return err
	}

	for _, networkName := range networkNames {
		if !strings.Contains(vmNetworkNames, networkName) {
			return fmt.Errorf("link %v is not added to VM", networkName)
		}
	}
	return nil
}
