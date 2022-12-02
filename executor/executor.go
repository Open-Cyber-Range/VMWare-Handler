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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
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
	Storage       *library.Storage
}

type guestManager struct {
	VirtualMachine *object.VirtualMachine
	Auth           *types.NamePasswordAuthentication
	ProcessManager *guest.ProcessManager
	FileManager    *guest.FileManager
}

type Feature struct {
	Type   string     `json:"type"`
	Action string     `json:"action,omitempty"`
	Assets [][]string `json:"assets"`
}

type GuestOSFamily int

const (
	Linux GuestOSFamily = iota
	Windows
)

var (
	SupportedOsFamilyMap = map[string]GuestOSFamily{
		"linuxGuest":   Linux,
		"windowsGuest": Windows,
	}
)

func parseOsFamily(vmOsFamily string) (family GuestOSFamily, success bool) {
	family, success = SupportedOsFamilyMap[vmOsFamily]
	return
}

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
		return status.Error(codes.Internal, fmt.Sprintf("Error while uploading file: %v", response.Status))
	} else {
		log.Printf("Successfully uploaded file to vm")
	}
	return err
}

func receiveFileFromVM(url string) (string, error) {
	out, err := os.CreateTemp("", "executor.log")
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error creating file, %v", err))
	}

	logPath := out.Name()
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
	return logPath, nil
}

func normalizeTargetPath(sourcePath string, destinationPath string) string {

	const unixPathSeperator = "/"
	const windowsPathSeperator = "\\"

	if strings.HasSuffix(destinationPath, unixPathSeperator) ||
		strings.HasSuffix(destinationPath, windowsPathSeperator) {
		destinationPath = strings.Join([]string{destinationPath, filepath.Base(sourcePath)}, "")
	}

	return destinationPath
}

func getFeatureInfo(packegeDataMap *map[string]interface{}) (feature Feature, err error) {
	featureInfo := (*packegeDataMap)["feature"]
	infoJson, err := json.Marshal(featureInfo)
	json.Unmarshal(infoJson, &feature)
	return
}

func createFileAttributesByOsFamily(guestOsFamily GuestOSFamily, filePermissions string) (fileAttributes types.BaseGuestFileAttributes, err error) {
	switch guestOsFamily {
	case Linux:
		permissionsInt, err := strconv.Atoi(filePermissions)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error converting str to int, %v", err))
		}
		fileAttributes = &types.GuestPosixFileAttributes{
			Permissions: int64(permissionsInt),
		}

	case Windows:
		// TODO: Implement application of Windows file permissions
		fileAttributes = &types.GuestWindowsFileAttributes{}
	}
	return fileAttributes, nil
}
func (guestManager *guestManager) findGuestOSFamily(ctx context.Context) (GuestOSFamily, error) {
	var vmProperties mo.VirtualMachine
	guestManager.VirtualMachine.Properties(ctx, guestManager.VirtualMachine.Reference(), []string{}, &vmProperties)

	matchedFamily, successful_match := parseOsFamily(vmProperties.Guest.GuestFamily)
	if successful_match {
		return matchedFamily, nil
	}

	return 0, status.Error(codes.Internal, "Guest OS Family not supported")
}

func (guestManager *guestManager) getVMLogContents(ctx context.Context, vmLogPath string) (output string, err error) {
	transferInfo, err := guestManager.FileManager.InitiateFileTransferFromGuest(ctx, guestManager.Auth, vmLogPath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error retrieving execution log from guest, %v", err))
	}

	filePath, err := receiveFileFromVM(transferInfo.Url)
	if err != nil {
		return "", err
	}

	logContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error opening log file, %v", err))
	}

	return string(logContent), nil
}

func (guestManager *guestManager) awaitProcessCompletion(ctx context.Context, processID int64) (bool, error) {
	for {
		log.Infof("Program running. PID: %v", processID)
		active_process, err := guestManager.ProcessManager.ListProcesses(ctx, guestManager.Auth, []int64{processID})
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

func (guestManager *guestManager) executePackageAction(ctx context.Context, action string) (vmLog string, err error) {
	vmLogPath, err := guestManager.FileManager.CreateTemporaryFile(ctx, guestManager.Auth, "executor-", ".log", "")
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error creating log file on vm, %v", err))

	}
	programSpec := &types.GuestProgramSpec{
		ProgramPath:      action,
		Arguments:        fmt.Sprintf("> %v 2>&1", vmLogPath),
		WorkingDirectory: path.Dir(action),
		EnvVariables:     []string{},
	}

	processID, err := guestManager.ProcessManager.StartProgram(ctx, guestManager.Auth, programSpec)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error starting program, %v", err))
	}

	if processID > 0 {
		err = guestManager.awaitProcessWithTimeout(ctx, processID, action)
		if err != nil {
			return
		}
	}
	vmLog, err = guestManager.getVMLogContents(ctx, vmLogPath)
	if err != nil {
		return
	}
	log.Infof("PID %v output: %v", processID, vmLog)
	return
}

func (server *featurerServer) createGuestManagers(ctx context.Context, featureDeployment *feature.Feature,
	auth *types.NamePasswordAuthentication) (*guestManager, error) {
	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, featureDeployment.VirtualMachineId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting VM by UUID, %v", err))
	}

	vmToolsRunning, err := library.CheckVMStatus(ctx, virtualMachine)
	if err != nil {
		return nil, err
	} else if !vmToolsRunning {
		err = vmwareClient.AwaitVMToolsToComeOnline(ctx, featureDeployment.VirtualMachineId)
		if err != nil {
			return nil, err
		}
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
		FileManager:    fileManager,
		ProcessManager: processManager,
	}
	return guestManager, nil
}

func (guestManager *guestManager) copyAssetsToVM(ctx context.Context, assets [][]string, packagePath string, storage *library.Storage, currentDeplyoment library.ExecutorContainer, executorId string) (err error) {
	for i := 0; i < len(assets); i++ {

		guestOsFamily, err := guestManager.findGuestOSFamily(ctx)
		if err != nil {
			return err
		}

		sourcePath := path.Join(packagePath, filepath.FromSlash(assets[i][0]))
		targetPath := assets[i][1]

		var filePermissions string
		if len(assets[i]) < 3 {
			filePermissions = ""
		} else {
			filePermissions = assets[i][2]
		}

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Error getting file information, %v", err))
		}

		fileAttributes, err := createFileAttributesByOsFamily(guestOsFamily, filePermissions)
		if err != nil {
			return err
		}

		normalizedTargetPath := normalizeTargetPath(sourcePath, targetPath)

		err = guestManager.FileManager.MakeDirectory(ctx, guestManager.Auth, path.Dir(normalizedTargetPath), true)
		if err != nil && !strings.HasSuffix(err.Error(), "already exists") {
			return status.Error(codes.Internal, fmt.Sprintf("Error creating VM directories, %v", err))
		}

		transferUrl, err := guestManager.FileManager.InitiateFileTransferToGuest(ctx, guestManager.Auth, normalizedTargetPath, fileAttributes, fileInfo.Size(), true)
		if err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Error creating transfer URL, %v", err))
		}

		if err = sendFileToVM(transferUrl, sourcePath); err != nil {
			return err
		}

		currentDeplyoment.FilePaths = append(currentDeplyoment.FilePaths, normalizedTargetPath)
		if err = library.Update(ctx, storage.RedisClient, executorId, &currentDeplyoment); err != nil {
			return err
		}
	}
	return nil
}

func (server *featurerServer) Create(ctx context.Context, featureDeployment *feature.Feature) (featureResponse *feature.FeatureResponse, err error) {

	accounts, err := library.Get(ctx, server.Storage.RedisClient, featureDeployment.TemplateId, new([]library.Account))
	if err != nil {
		return nil, err
	}
	log.Infof("Got %v account(s) for current VM", len(*accounts))

	var password string
	for _, account := range *accounts {
		if account.Name == featureDeployment.Username {
			password = account.Password
		}
	}
	auth := &types.NamePasswordAuthentication{
		Username: featureDeployment.Username,
		Password: password,
	}

	guestManager, err := server.createGuestManagers(ctx, featureDeployment, auth)
	if err != nil {
		return nil, err
	}

	_, err = library.CheckVMStatus(ctx, guestManager.VirtualMachine)
	if err != nil {
		return nil, err
	}

	packagePath, err := library.DownloadPackage(featureDeployment.GetSource().GetName(), featureDeployment.GetSource().GetVersion())
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package (%v)", err))
	}

	packageTomlContent, err := library.GetPackageData(packagePath)
	if err != nil {
		log.Errorf("Failed to get package data, %v", err)
	}

	packageFeature, err := getFeatureInfo(&packageTomlContent)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting package info, %v", err))
	}

	featureId := uuid.New().String()

	currentDeplyoment := library.ExecutorContainer{
		VMID:      featureDeployment.VirtualMachineId,
		Auth:      *guestManager.Auth,
		FilePaths: []string{},
	}

	library.Create(ctx, server.Storage.RedisClient, featureId, currentDeplyoment)

	if err = guestManager.copyAssetsToVM(ctx, packageFeature.Assets, packagePath, server.Storage, currentDeplyoment, featureId); err != nil {
		return nil, err
	}

	//temp assignment of Action field for test TestFeatureDeploymentAndDeletionOnLinux since Deputy is unreleased
	packageFeature.Action = "/tmp/test-folder/todays_date.sh"

	var vmLog string
	if packageFeature.Action != "" && featureDeployment.FeatureType == *feature.FeatureType_service.Enum() {
		vmLog, err = guestManager.executePackageAction(ctx, packageFeature.Action)
		if err != nil {
			return nil, err
		}
	}

	featureResponse = &feature.FeatureResponse{
		Identifier: &common.Identifier{
			Value: fmt.Sprintf("%v", featureId),
		},
		VmLog: vmLog,
	}

	return
}

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {

	featureContainer, err := library.Get(ctx, server.Storage.RedisClient, identifier.GetValue(), new(library.ExecutorContainer))
	if err != nil {
		return nil, err
	}

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
		err = fileManager.DeleteFile(ctx, &featureContainer.Auth, string(targetFile))
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

	redisClient := library.NewStorage(redis.NewClient(&redis.Options{
		Addr:     configuration.RedisAddress,
		Password: configuration.RedisPassword,
		DB:       0,
	}))
	server := grpc.NewServer()

	feature.RegisterFeatureServiceServer(server, &featurerServer{
		Client:        client,
		Configuration: configuration,
		Storage:       &redisClient,
	})

	capabilityServer := library.NewCapabilityServer([]capability.Capabilities_DeployerTypes{
		*capability.Capabilities_Feature.Enum(),
	})

	capability.RegisterCapabilityServer(server, &capabilityServer)

	log.Printf("Executor listening at %v", listeningAddress.Addr())
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
