package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"path"
	"strings"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/template"
	"github.com/open-cyber-range/vmware-handler/library"
	"github.com/vmware/govmomi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type templaterServer struct {
	template.UnimplementedTemplateServiceServer
	Client        *govmomi.Client
	Configuration library.Configuration
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
	log.Printf("Created package base path: %v\n", packageBasePath)

	downloadCommand := exec.Command("deputy", "fetch", templateDeployment.source.Name, "-v", templateDeployment.source.Version, "-s", packagePath)
	downloadCommand.Dir = packageBasePath
	downloadCommand.Run()
	directories, err := library.IOReadDir(packageBasePath)
	if err != nil {
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

func (templateDeployment *TemplateDeployment) getPackageData(packagePath string) (packageData map[string]interface{}, err error) {
	packageTomlPath := path.Join(packagePath, "package.toml")
	checksumCommand := exec.Command("deputy", "info", packageTomlPath)
	output, err := checksumCommand.Output()
	if err != nil {
		return
	}
	json.Unmarshal(output, &packageData)
	return
}

type VirtualMachine struct {
	OperatingSystem string `json:"operating_system,omitempty"`
	Architecture    string `json:"architecture,omitempty"`
	Type            string `json:"type,omitempty"`
	FilePath        string `json:"file_path,omitempty"`
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

func (server *templaterServer) Create(ctx context.Context, source *common.Source) (*common.Identifier, error) {
	log.Printf("Received template package: %v, version: %v\n", source.Name, source.Version)

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	checksum, checksumError := getPackageChecksum(source.Name, source.Version)
	if checksumError != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: failed to get package checksum (%v)", checksumError))
		return nil, checksumError
	}

	templateName := fmt.Sprintf("%v-%v-%v", source.Name, source.Version, checksum)
	templateExists, templateExistsError := vmwareClient.DoesTemplateExist(templateName)
	if templateExistsError != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: failed to get information about template from VSphere(%v)", templateExistsError))
		return nil, templateExistsError
	}

	if templateExists {
		log.Printf("Template already deployed: %v, version: %v\n", source.Name, source.Version)
	} else {
		templateDeployment := TemplateDeployment{
			Client:        &vmwareClient,
			source:        source,
			Configuration: server.Configuration,
			templateName:  templateName,
		}
		packagePath, downloadError := templateDeployment.downloadPackage()
		if downloadError != nil {
			status.New(codes.NotFound, fmt.Sprintf("Create: failed to download package (%v)", downloadError))
			return nil, downloadError
		}
		log.Printf("Downloaded package to: %v\n", packagePath)

		deployError := templateDeployment.createTemplate(packagePath)
		if deployError != nil {
			status.New(codes.Internal, fmt.Sprintf("Create: failed to deploy template (%v)", deployError))
			return nil, deployError
		}
		log.Printf("Deployed template: %v, version: %v\n", source.Name, source.Version)
	}
	deployedTemplate, deloyedTemplateError := vmwareClient.GetTemplateByName(templateName)
	if deloyedTemplateError != nil {
		status.New(codes.Internal, fmt.Sprintf("Create: failed to get deployed template (%v)", deloyedTemplateError))
		return nil, deloyedTemplateError
	}

	status.New(codes.OK, "Node creation successful")
	return &common.Identifier{
		Value: deployedTemplate.UUID(ctx),
	}, nil
}

func RealMain(configuration library.Configuration) {
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
	template.RegisterTemplateServiceServer(server, &templaterServer{
		Client:        client,
		Configuration: configuration,
	})
	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Template.Enum().Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("server listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("failed to serve: %v", bindError)
	}
}

func main() {
	log.SetPrefix("templater: ")
	log.SetFlags(0)

	configuration, configurationError := library.NewValidator().SetRequireDatastorePath(true).GetConfiguration()

	if configurationError != nil {
		log.Fatal(configurationError)
	}
	log.Println("Hello, templater!")
	RealMain(configuration)
}
