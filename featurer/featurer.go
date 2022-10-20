package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type featurerServer struct {
	feature.UnimplementedFeatureServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
}

type featureContainer struct {
	VMID      string
	Auth      types.BaseGuestAuthentication
	FilePaths []string
}

type guestManager struct {
	VirtualMachine *object.VirtualMachine
	Auth           types.BaseGuestAuthentication
	Processes      *guest.ProcessManager
	Files          *guest.FileManager
}

type Feature struct {
	Assets [][]string `json:"assets"`
}

var deployedFeatures = make(map[string]*featureContainer)

func sendFileToVM(url string, filePath string) error {
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := http.Client{Transport: transport, Timeout: 30 * time.Second}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error reading source file, %v", err))
	}

	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(file))
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error creating HTTP request, %v", err))
	}

	response, err := client.Do(request)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error sending HTTP request: %v", err))
	}
	if response.StatusCode != 200 {
		return status.Error(codes.Internal, fmt.Sprintf("Error while uploading file: %v", response.StatusCode))
	} else {
		log.Printf("Successfully uploaded file: %v", filePath)
	}
	return err
}

func receiveFileFromVM(url string, filePath string) (string, error) {
	out, err := os.Create(filePath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error creating file, %v", err))
	}
	defer out.Close()

	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := http.Client{Transport: transport, Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error sending HTTP request, %v", err))
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func normalizeTargetPath(sourcePath string, destinationPath string) string {
	if strings.HasSuffix(destinationPath, "/") {
		sourcePathSlices := strings.Split(sourcePath, "/")
		sourceFileName := sourcePathSlices[len(sourcePathSlices)-1]
		destinationPath = strings.Join([]string{destinationPath, sourceFileName}, "")
	}
	return destinationPath
}

func getFeatureInfo(packegeDataMap *map[string]interface{}) (feature Feature, err error) {
	featureInfo := (*packegeDataMap)["feature"]
	infoJson, err := json.Marshal(featureInfo)
	json.Unmarshal(infoJson, &feature)
	return
}

func (guestManager *guestManager) getVMLogContents(ctx context.Context, logPath string) (output string, err error) {
	transferInfo, err := guestManager.Files.InitiateFileTransferFromGuest(ctx, guestManager.Auth, logPath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error retrieving execution log from guest, %v", err))
	}

	filePath, err := receiveFileFromVM(transferInfo.Url, logPath)
	if err != nil {
		return "", err
	}

	logContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error opening log file, %v", err))
	}

	return string(logContent), nil
}

func (guestManager *guestManager) applyPermissionsToFile(ctx context.Context, filePath string, permissions string) (err error) {

	programSpec := &types.GuestProgramSpec{
		ProgramPath: "/usr/bin/chmod",
		Arguments:   fmt.Sprintf("%v %v &> /tmp/featurer.log", permissions, filePath),
	}
	_, err = guestManager.Processes.StartProgram(ctx, guestManager.Auth, programSpec)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return
}

func (guestManager *guestManager) awaitProcessCompletion(ctx context.Context, processID int64) (bool, error) {
	for {
		log.Infof("Program running. PID: %v", processID)
		active_process, err := guestManager.Processes.ListProcesses(ctx, guestManager.Auth, []int64{processID})
		if err != nil {
			return false, status.Error(codes.Internal, fmt.Sprintf("Error listing VM process' %v", err))
		}

		time.Sleep(2000 * time.Millisecond)

		exitCode := active_process[0].ExitCode
		regexFail, _ := regexp.Compile("[1-9]+")
		if regexFail.MatchString(string(exitCode)) {
			return false, status.Error(codes.Internal, fmt.Sprintf("Program encountered an error, code: %v", exitCode))

		}

		processEndTime := active_process[0].EndTime
		regexSuccess, _ := regexp.Compile("[^0-9]+")
		successCheck := regexSuccess.MatchString(string(exitCode))
		if successCheck && processEndTime != nil {
			log.Infof("Program %v completed with success", processID)
			break
		}
	}
	return true, nil
}
func (guestManager *guestManager) awaitProcessWithTimeout(ctx context.Context, processID int64, normalizedTargetPath string) error {
	resultChannel := make(chan bool)
	errorChannel := make(chan error)

	timeout := 300 * time.Second

	go func() {
		processFinished, err := guestManager.awaitProcessCompletion(ctx, processID)
		resultChannel <- processFinished
		errorChannel <- err
	}()
	select {
	case processSucceeded := <-resultChannel:
		if !processSucceeded {
			err := <-errorChannel
			if err != nil {
				return err
			}
		}
	case <-time.After(timeout):
		log.Errorf("Process %v timed out after %v seconds", normalizedTargetPath, timeout)
	}
	return nil
}

func (server *featurerServer) createGuestManagers(ctx context.Context, featureDeployment *feature.Feature, auth types.BaseGuestAuthentication) (*guestManager, error) {
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, featureDeployment.VirtualMachineId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting VM by UUID, %v", err))
	}
	operationsManager := guest.NewOperationsManager(virtualMachine.Client(), virtualMachine.Reference())
	fileManager, err := operationsManager.FileManager(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating FileManager, %v", err))
	}

	processManager, err := operationsManager.ProcessManager(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating a process manager, %v", err))
	}

	guestManager := &guestManager{
		VirtualMachine: virtualMachine,
		Auth:           auth,
		Files:          fileManager,
		Processes:      processManager,
	}
	return guestManager, nil
}

func (server *featurerServer) Create(ctx context.Context, featureDeployment *feature.Feature) (identifier *common.Identifier, err error) {

	packagePath, err := library.DownloadPackage(featureDeployment.GetSource().GetName(), featureDeployment.GetSource().GetVersion())
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package (%v)", err))
	}
	log.Infof("Downloaded package to: %v", packagePath)

	packageTomlContent, err := library.GetPackageData(packagePath)
	if err != nil {
		log.Errorf("Failed to get package data, %v", err)
	}
	log.Infof("the entire package %v", packageTomlContent)

	packageFeature, err := getFeatureInfo(&packageTomlContent)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting package info, %v", err))
	}

	featureId := uuid.New().String()
	auth := &types.NamePasswordAuthentication{
		Username: featureDeployment.User,
		Password: featureDeployment.Password,
	}

	guestManager, err := server.createGuestManagers(ctx, featureDeployment, auth)
	if err != nil {
		return nil, err
	}

	currentDeplyoment := &featureContainer{
		VMID:      featureDeployment.VirtualMachineId,
		Auth:      guestManager.Auth,
		FilePaths: []string{},
	}
	deployedFeatures[featureId] = currentDeplyoment

	for i := 0; i < len(packageFeature.Assets); i++ {

		sourcePath := path.Join(packagePath, packageFeature.Assets[i][0])
		targetPath := packageFeature.Assets[i][1]

		// TODO: Set log and target file path formats depetnding on guest os
		vmLogPath := "/tmp/featurer.log"
		normalizedTargetPath := normalizeTargetPath(sourcePath, targetPath)

		var filePermissions string
		if len(packageFeature.Assets[i]) < 3 {
			filePermissions = ""
		} else {
			filePermissions = packageFeature.Assets[i][2]
		}

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting file information, %v", err))
		}

		err = guestManager.Files.MakeDirectory(ctx, auth, path.Dir(normalizedTargetPath), true)
		if err != nil && !strings.HasSuffix(err.Error(), "already exists") {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating VM directories, %v", err))
		}

		transferUrl, err := guestManager.Files.InitiateFileTransferToGuest(ctx, auth, normalizedTargetPath, &types.GuestFileAttributes{}, fileInfo.Size(), true)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating transfer URL, %v", err))
		}

		err = sendFileToVM(transferUrl, sourcePath)
		if err != nil {
			return nil, err
		}

		if filePermissions != "" {
			err = guestManager.applyPermissionsToFile(ctx, normalizedTargetPath, filePermissions)
			if err != nil {
				return nil, err
			}

			permissionsOutput, err := guestManager.getVMLogContents(ctx, vmLogPath)
			if err != nil {
				return nil, err
			} else if permissionsOutput != "" {
				return nil, status.Error(codes.Internal, fmt.Sprintf("Error applying permissions to file, %v", permissionsOutput))

			}
		} else {
			log.Infof("No file permissions assigned for %v, skipping to next step", sourcePath)
		}

		if featureDeployment.FeatureType == *feature.FeatureType_service.Enum() {
			programSpec := &types.GuestProgramSpec{
				ProgramPath:      normalizedTargetPath,
				Arguments:        fmt.Sprintf("-b -n &> %v", vmLogPath),
				WorkingDirectory: path.Dir(normalizedTargetPath),
			}

			processID, err := guestManager.Processes.StartProgram(ctx, auth, programSpec)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
			}

			if processID > 0 {
				err := guestManager.awaitProcessWithTimeout(ctx, processID, normalizedTargetPath)
				if err != nil {
					return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
				}
			}
			output, err := guestManager.getVMLogContents(ctx, vmLogPath)
			if err != nil {
				return nil, err
			}
			log.Infof("PID %v output: %v", processID, output)

		}
		deployedFeatures[featureId].FilePaths = append(deployedFeatures[featureId].FilePaths, normalizedTargetPath)

	}
	identifier = &common.Identifier{
		Value: fmt.Sprintf("%v", featureId),
	}
	return
}

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	featureContainer := deployedFeatures[identifier.Value]

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, featureContainer.VMID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting VM by UUID, %v", err))
	}
	operationsManager := guest.NewOperationsManager(virtualMachine.Client(), virtualMachine.Reference())

	fileManager, err := operationsManager.FileManager(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating FileManager, %v", err))
	}
	for i := 0; i < len(featureContainer.FilePaths); i++ {

		targetFile := featureContainer.FilePaths[i]
		err = fileManager.DeleteFile(ctx, featureContainer.Auth, string(targetFile))
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error deleting file, %v", err))
		}
		log.Infof("Deleted %v", targetFile)

		time.Sleep(200 * time.Millisecond)
	}
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
		log.Fatalf("Failed to listen: %v", addressError)
	}

	server := grpc.NewServer()
	feature.RegisterFeatureServiceServer(server, &featurerServer{
		Client:        client,
		Configuration: configuration,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Feature.Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("Featurer listening at %v", listeningAddress.Addr())
	if bindError := server.Serve(listeningAddress); bindError != nil {
		log.Fatalf("Failed to serve: %v", bindError)
	}
}

func main() {
	configuration, configurationError := library.NewValidator().SetRequireExerciseRootPath(true).GetConfiguration()
	if configurationError != nil {
		log.Fatal(configurationError)
	}
	RealMain(&configuration)
}
