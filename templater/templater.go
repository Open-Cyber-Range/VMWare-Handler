package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/template"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeploymentList struct {
	sync.RWMutex
	items []string
}

func (deploymentList *DeploymentList) Append(item string) {
	deploymentList.Lock()
	defer deploymentList.Unlock()

	deploymentList.items = append(deploymentList.items, item)
}

func (deploymentList *DeploymentList) DeploymentExists(searchItem string) bool {
	deploymentList.Lock()
	defer deploymentList.Unlock()

	for _, value := range deploymentList.items {
		if value == searchItem {
			return true
		}
	}

	return false
}

func (deploymentList *DeploymentList) WaitForDeployment(deploymentName string) {
	waitChannel := make(chan bool)
	defer close(waitChannel)

	go func() {
		for deploymentList.DeploymentExists(deploymentName) {
			time.Sleep(time.Second)
		}

		waitChannel <- true
	}()
	<-waitChannel
}

func (deploymentList *DeploymentList) Remove(deleteItem string) error {
	deploymentList.Lock()
	defer deploymentList.Unlock()

	searchIndex := -1
	for index, value := range deploymentList.items {
		if value == deleteItem {
			searchIndex = index
		}
	}

	if searchIndex == -1 {
		return fmt.Errorf("deployment %v not found", deleteItem)
	}

	deploymentList.items = append(deploymentList.items[:searchIndex], deploymentList.items[searchIndex+1:]...)
	return nil
}

type templaterServer struct {
	template.UnimplementedTemplateServiceServer
	Client                *govmomi.Client
	Configuration         library.Configuration
	currentDeploymentList DeploymentList
	Storage               *library.Storage
}

type TemplateDeployment struct {
	Client        *library.VMWareClient
	source        *common.Source
	Configuration library.Configuration
	templateName  string
	metaInfo      string
	Storage       *library.Storage
}

type Account struct {
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

type VirtualMachine struct {
	OperatingSystem string    `json:"operating_system,omitempty"`
	Architecture    string    `json:"architecture,omitempty"`
	Type            string    `json:"type,omitempty"`
	FilePath        string    `json:"file_path,omitempty"`
	Links           []string  `json:"links,omitempty"`
	Accounts        []Account `json:"accounts,omitempty"`
}

func getVirtualMachineInfo(packegeDataMap *map[string]interface{}) (virtualMachine VirtualMachine, err error) {
	virtualMachineInfo := (*packegeDataMap)["virtual-machine"]
	infoJson, _ := json.Marshal(virtualMachineInfo)
	json.Unmarshal(infoJson, &virtualMachine)

	return
}

func (templateDeployment *TemplateDeployment) saveTemplateAccounts(ctx context.Context, templateId string, virtualMachine VirtualMachine) (err error) {
	err = library.Create(ctx, templateDeployment.Storage.RedisClient, templateId, virtualMachine.Accounts)

	return
}

func (templateDeployment *TemplateDeployment) handleTemplateBasedOnType(virtualMachine VirtualMachine, packagePath string) (err error) {
	switch virtualMachine.Type {
	case "OVA":
		_, err = templateDeployment.ImportOVA(path.Join(packagePath, virtualMachine.FilePath), templateDeployment.Client.Client.Client)
	}
	return
}

func (templateDeployment *TemplateDeployment) createTemplate(ctx context.Context, packagePath string) (err error) {
	packageData, err := library.GetPackageData(packagePath)
	if err != nil {
		return
	}
	virtualMachine, err := getVirtualMachineInfo(&packageData)
	if err != nil {
		return
	}
	if err = templateDeployment.handleTemplateBasedOnType(virtualMachine, packagePath); err != nil {
		return
	}
	deployedTemplate, err := templateDeployment.Client.GetTemplateByName(templateDeployment.templateName)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Failed to get deployed template (%v)", err))
	}

	err = templateDeployment.saveTemplateAccounts(ctx, deployedTemplate.UUID(ctx), virtualMachine)

	return
}

func tryRemoveNetworks(ctx context.Context, deployedTemplate *object.VirtualMachine) {
	devices, devicesError := deployedTemplate.Device(ctx)
	if devicesError != nil {
		log.Errorf("Error getting devices: %v", devicesError)
	}

	networkDevices := devices.SelectByType(&types.VirtualEthernetCard{})

	for _, networkDevice := range networkDevices {
		networkRemovalError := deployedTemplate.RemoveDevice(ctx, false, networkDevice)
		if networkRemovalError != nil {
			log.Errorf("Failed to remove network: %v", networkRemovalError)
		}
	}
}

func (server *templaterServer) Create(ctx context.Context, source *common.Source) (*common.Identifier, error) {
	log.Infof("Received template package: %v, version: %v", source.Name, source.Version)

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	checksum, checksumError := library.GetPackageChecksum(source.Name, source.Version)
	if checksumError != nil {
		log.Errorf("Error getting package checksum: %v", checksumError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to get package checksum (%v)", checksumError))
	}
	normalizedVersion, normalizedVersionError := library.NormalizePackageVersion(source.Name, source.Version)
	if normalizedVersionError != nil {
		log.Errorf("Failed to normalize package version (%v)", normalizedVersionError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to normalize package version (%v)", normalizedVersionError))
	}

	templateName := checksum
	templateExists, templateExistsError := vmwareClient.DoesTemplateExist(templateName)
	if templateExistsError != nil {
		log.Errorf("Failed to check if template exists (%v)", templateExistsError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to get information about template from VSphere(%v)", templateExistsError))
	}

	if templateExists {
		log.Infof("Template already deployed: %v, version: %v", source.Name, source.Version)
	} else if server.currentDeploymentList.DeploymentExists(templateName) {
		log.Infof("Template is being deployed: %v, version: %v", source.Name, source.Version)
		server.currentDeploymentList.WaitForDeployment(templateName)
	} else {
		server.currentDeploymentList.Append(templateName)
		templateDeployment := TemplateDeployment{
			Client:        &vmwareClient,
			source:        source,
			Configuration: server.Configuration,
			templateName:  templateName,
			metaInfo:      fmt.Sprintf("%v-%v", source.Name, normalizedVersion),
			Storage:       server.Storage,
		}
		packagePath, downloadError := library.DownloadPackage(templateDeployment.source.Name, templateDeployment.source.Version)
		if downloadError != nil {
			log.Errorf("Failed to download package (%v)", downloadError)
			return nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package (%v)", downloadError))
		}
		log.Infof("Downloaded package to: %v", packagePath)

		deployError := templateDeployment.createTemplate(ctx, packagePath)
		server.currentDeploymentList.Remove(templateName)
		if deployError != nil {
			log.Printf("Failed to deploy template (%v)", deployError)
			return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to deploy template (%v)", deployError))
		}
		log.Infof("Deployed template: %v, version: %v", source.Name, source.Version)
	}

	deployedTemplate, deloyedTemplateError := vmwareClient.GetTemplateByName(templateName)
	if deloyedTemplateError != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to get deployed template (%v)", deloyedTemplateError))
	}
	tryRemoveNetworks(ctx, deployedTemplate)

	return &common.Identifier{
		Value: deployedTemplate.UUID(ctx),
	}, nil
}

func (server *templaterServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	uuid := identifier.GetValue()

	virtualMachine, virtualMachineError := vmwareClient.GetVirtualMachineByUUID(ctx, uuid)
	if virtualMachineError != nil {
		log.Errorf("Template not found error (%v)", virtualMachineError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Template not found error (%v)", virtualMachineError))
	}
	templateName, templateNameError := virtualMachine.ObjectName(ctx)
	if templateNameError != nil {
		log.Errorf("Template name retrieval error (%v)", templateNameError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Template name retrieval error (%v)", templateNameError))
	}
	log.Infof("Received node for deleting: %v with UUID: %v", templateName, uuid)

	deploymentError := vmwareClient.DeleteVirtualMachineByUUID(uuid)
	if deploymentError != nil {
		log.Errorf("Failed to delete node: %v", deploymentError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during deletion (%v)", deploymentError))
	}

	log.Infof("Deleted template: %v", templateName)
	return new(emptypb.Empty), nil
}

func RealMain(configuration library.Configuration) {
	ctx := context.Background()
	client, clientError := configuration.CreateClient(ctx)
	if clientError != nil {
		log.Fatal(clientError)
	}

	listeningAddress, addressError := net.Listen("tcp", configuration.ServerAddress)
	if addressError != nil {
		log.Fatalf("Failed to listen: %v", addressError)
	}
	redisClient := library.NewStorage(redis.NewClient(&redis.Options{
		Addr:     configuration.RedisAddress,
		Password: configuration.RedisPassword,
		DB:       0,
	}))

	server := grpc.NewServer()
	template.RegisterTemplateServiceServer(server, &templaterServer{
		Client:        client,
		Configuration: configuration,
		Storage:       &redisClient,
	})
	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Template.Enum().Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Infof("Templater listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	configuration, configurationError := library.NewValidator().SetRequireDatastorePath(true).GetConfiguration()

	if configurationError != nil {
		log.Fatal(configurationError)
	}
	RealMain(configuration)
}
