package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

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
}

type TemplateDeployment struct {
	Client        *library.VMWareClient
	source        *common.Source
	Configuration library.Configuration
	templateName  string
}

func createRandomPackagePath() (string, error) {
	return ioutil.TempDir("/tmp", "deputy-package")
}

func (templateDeployment *TemplateDeployment) downloadPackage() (packagePath string, err error) {
	packageBasePath, err := createRandomPackagePath()
	if err != nil {
		return
	}
	log.Printf("Created package base path: %v", packageBasePath)

	downloadCommand := exec.Command("deputy", "fetch", templateDeployment.source.Name, "-v", templateDeployment.source.Version, "-s", packagePath)
	downloadCommand.Dir = packageBasePath
	_, err = downloadCommand.Output()
	if err != nil {
		return
	}
	directories, err := library.IOReadDir(packageBasePath)
	if err != nil {
		return
	}

	if len(directories) != 1 {
		err = fmt.Errorf("expected one directory in package base path, got %v", len(directories))
		return
	}

	return path.Join(packageBasePath, directories[0]), nil
}

func getPackageChecksum(name string, version string) (checksum string, err error) {
	checksumCommand := exec.Command("deputy", "checksum", name, "-v", version)
	output, err := checksumCommand.Output()
	if err != nil {
		return
	}
	checksum = strings.TrimSpace(string(output))
	return
}

func normalizePackageVersion(packageName string, versionRequirement string) (normalizedVersion string, err error) {
	versionCommand := exec.Command("deputy", "normalize-version", packageName, "-v", versionRequirement)
	output, err := versionCommand.Output()
	if err != nil {
		return
	}
	normalizedVersion = strings.TrimSpace(string(output))
	return
}

func (templateDeployment *TemplateDeployment) getPackageData(packagePath string) (packageData map[string]interface{}, err error) {
	packageTomlPath := path.Join(packagePath, "package.toml")
	checksumCommand := exec.Command("deputy", "parse-toml", packageTomlPath)
	output, err := checksumCommand.Output()
	if err != nil {
		return
	}
	json.Unmarshal(output, &packageData)
	return
}

type VirtualMachine struct {
	OperatingSystem string   `json:"operating_system,omitempty"`
	Architecture    string   `json:"architecture,omitempty"`
	Type            string   `json:"type,omitempty"`
	FilePath        string   `json:"file_path,omitempty"`
	Links           []string `json:"links,omitempty"`
}

func getVirtualMachineInfo(packegeDataMap *map[string]interface{}) (virtualMachine VirtualMachine, err error) {
	virtualMachineInfo := (*packegeDataMap)["virtual-machine"]
	infoJson, _ := json.Marshal(virtualMachineInfo)
	json.Unmarshal(infoJson, &virtualMachine)

	return
}

func (templateDeployment *TemplateDeployment) handleTemplateBasedOnType(packageData map[string]interface{}, packagePath string) (err error) {
	virtualMachine, err := getVirtualMachineInfo(&packageData)
	if err != nil {
		return
	}
	switch virtualMachine.Type {
	case "OVA":
		_, err = templateDeployment.ImportOVA(path.Join(packagePath, virtualMachine.FilePath), templateDeployment.Client.Client.Client)
	}
	return
}

func (templateDeployment *TemplateDeployment) createTemplate(packagePath string) (err error) {
	packageData, err := templateDeployment.getPackageData(packagePath)
	if err != nil {
		return
	}
	err = templateDeployment.handleTemplateBasedOnType(packageData, packagePath)

	return
}

func tryRemoveNetworks(ctx context.Context, deployedTemplate *object.VirtualMachine) {
	devices, devicesError := deployedTemplate.Device(ctx)
	if devicesError != nil {
		log.Printf("Error getting devices: %v", devicesError)
	}

	networkDevices := devices.SelectByType(&types.VirtualEthernetCard{})

	for _, networkDevice := range networkDevices {
		networkRemovalError := deployedTemplate.RemoveDevice(ctx, false, networkDevice)
		if networkRemovalError != nil {
			log.Printf("Failed to remove network: %v", networkRemovalError)
		}
	}
}

func (server *templaterServer) Create(ctx context.Context, source *common.Source) (*common.Identifier, error) {
	log.Printf("Received template package: %v, version: %v", source.Name, source.Version)

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	checksum, checksumError := getPackageChecksum(source.Name, source.Version)
	if checksumError != nil {
		log.Printf("Error getting package checksum: %v", checksumError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to get package checksum (%v)", checksumError))
	}
	normalizedVersion, normalizedVersionError := normalizePackageVersion(source.Name, source.Version)
	if normalizedVersionError != nil {
		log.Printf("Failed to normalize package version (%v)", normalizedVersionError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to normalize package version (%v)", normalizedVersionError))
	}

	templateName := fmt.Sprintf("%v-%v-%v", source.Name, normalizedVersion, checksum)
	templateExists, templateExistsError := vmwareClient.DoesTemplateExist(templateName)
	if templateExistsError != nil {
		log.Printf("Failed to check if template exists (%v)", templateExistsError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to get information about template from VSphere(%v)", templateExistsError))
	}

	if templateExists {
		log.Printf("Template already deployed: %v, version: %v", source.Name, source.Version)
	} else if server.currentDeploymentList.DeploymentExists(templateName) {
		log.Printf("Template is being deployed: %v, version: %v", source.Name, source.Version)
		server.currentDeploymentList.WaitForDeployment(templateName)
	} else {
		server.currentDeploymentList.Append(templateName)
		templateDeployment := TemplateDeployment{
			Client:        &vmwareClient,
			source:        source,
			Configuration: server.Configuration,
			templateName:  templateName,
		}
		packagePath, downloadError := templateDeployment.downloadPackage()
		if downloadError != nil {
			log.Printf("Failed to download package (%v)", downloadError)
			return nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package (%v)", downloadError))
		}
		log.Printf("Downloaded package to: %v", packagePath)
		deployError := templateDeployment.createTemplate(packagePath)
		server.currentDeploymentList.Remove(templateName)
		if deployError != nil {
			log.Printf("Failed to deploy template (%v)", deployError)
			return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to deploy template (%v)", deployError))
		}
		log.Printf("Deployed template: %v, version: %v", source.Name, source.Version)
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
		log.Printf("Template not found error (%v)", virtualMachineError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Template not found error (%v)", virtualMachineError))
	}
	templateName, templateNameError := virtualMachine.ObjectName(ctx)
	if templateNameError != nil {
		log.Printf("Template name retrieval error (%v)", templateNameError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Template name retrieval error (%v)", templateNameError))
	}

	log.Printf("Received node for deleting: %v with UUID: %v", templateName, uuid)

	deploymentError := vmwareClient.DeleteVirtualMachineByUUID(uuid)
	if deploymentError != nil {
		log.Printf("Failed to delete node: %v", deploymentError)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during deletion (%v)", deploymentError))
	}

	log.Printf("Deleted template: %v", templateName)
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

	server := grpc.NewServer()
	template.RegisterTemplateServiceServer(server, &templaterServer{
		Client:        client,
		Configuration: configuration,
	})
	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Template.Enum().Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("Server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	configuration, configurationError := library.NewValidator().SetRequireDatastorePath(true).GetConfiguration()

	if configurationError != nil {
		log.Fatal(configurationError)
	}
	log.Println("Templater has started")
	RealMain(configuration)
}
