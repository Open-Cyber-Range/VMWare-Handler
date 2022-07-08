package library

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
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
