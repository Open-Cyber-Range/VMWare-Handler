package library

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/common"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/guest/toolbox"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"
)

type VMWareClient struct {
	Client       *govmomi.Client
	templatePath string
}

type GuestManager struct {
	VirtualMachine *object.VirtualMachine
	Auth           *types.NamePasswordAuthentication
	ProcessManager *guest.ProcessManager
	FileManager    *guest.FileManager
	Toolbox        *toolbox.Client
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

func NewVMWareClient(client *govmomi.Client, templatePath string) (configuration VMWareClient) {
	return VMWareClient{
		Client:       client,
		templatePath: templatePath,
	}
}

func (vmwareClient *VMWareClient) CreateGuestManagers(ctx context.Context, vmId string, auth *types.NamePasswordAuthentication) (*GuestManager, error) {
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, vmId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting VM by UUID, %v", err))
	}

	isVmToolsRunning, err := CheckVMStatus(ctx, virtualMachine)
	if err != nil {
		return nil, err
	} else if !isVmToolsRunning {
		err = AwaitVMToolsToComeOnline(ctx, virtualMachine)
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

	var vmManagedObject mo.VirtualMachine
	virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &vmManagedObject)
	toolboxClient, err := toolbox.NewClient(ctx, processManager.Client(), vmManagedObject.Reference(), auth)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating Toolbox Client' %v", err))
	}

	guestManager := &GuestManager{
		VirtualMachine: virtualMachine,
		Auth:           auth,
		FileManager:    fileManager,
		ProcessManager: processManager,
		Toolbox:        toolboxClient,
	}
	return guestManager, nil
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

	managedObjectList, err := finder.ManagedObjectList(ctx, client.templatePath)
	if err != nil {
		return nil, err
	}

	if len(managedObjectList) == 0 {
		return []*object.VirtualMachine{}, nil
	} else {
		return finder.VirtualMachineList(ctx, client.templatePath)
	}
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

func (vmwareClient *VMWareClient) DeleteUploadedFiles(ctx context.Context, executorContainer *ExecutorContainer) error {
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, executorContainer.VMID)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error getting VM by UUID: %v", err))
	}
	operationsManager := guest.NewOperationsManager(virtualMachine.Client(), virtualMachine.Reference())

	fileManager, err := operationsManager.FileManager(ctx)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error creating FileManager: %v", err))
	}

	for _, targetFile := range executorContainer.FilePaths {
		if err = fileManager.DeleteFile(ctx, &executorContainer.Auth, string(targetFile)); err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Error deleting file: %v", err))
		}
		log.Infof("Deleted %v", targetFile)
	}
	return nil
}

func normalizePackageTargetPath(sourcePath string, destinationPath string) string {

	const unixPathSeparator = "/"
	const windowsPathSeparator = "\\"

	if strings.HasSuffix(destinationPath, unixPathSeparator) ||
		strings.HasSuffix(destinationPath, windowsPathSeparator) {
		destinationPath = strings.Join([]string{destinationPath, filepath.Base(sourcePath)}, "")
	}

	return destinationPath
}

func sendFileToVM(url string, filePath string) (err error) {
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
	}
	return err
}

func receiveFileFromVM(url string) (logPath string, err error) {
	out, err := os.CreateTemp("", "executor.log")
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error creating file, %v", err))
	}

	logPath = out.Name()
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

func (guestManager *GuestManager) FindGuestOSFamily(ctx context.Context) (GuestOSFamily, error) {
	var vmProperties mo.VirtualMachine

	var tries int
	for tries = 0; tries < 60; tries++ {
		err := guestManager.VirtualMachine.Properties(ctx, guestManager.VirtualMachine.Reference(), []string{}, &vmProperties)
		if err != nil {
			return 0, status.Error(codes.Internal, fmt.Sprintf("Error retrieving VM properties, %v", err))
		}
		if vmProperties.Guest.GuestFamily == "" || vmProperties.Guest.GuestFamily == "0" {
			log.Info("Awaiting VM properties to be populated...") // change me to a debug log
			time.Sleep(1 * time.Second)
			continue

		} else {
			break
		}

	}
	if tries >= 60 {
		return 0, status.Error(codes.Internal, fmt.Sprintf("Timeout retrieving VM %v properties", guestManager.VirtualMachine.UUID(ctx)))
	}

	matchedFamily, successful_match := parseOsFamily(vmProperties.Guest.GuestFamily)
	if successful_match {
		return matchedFamily, nil
	} else {
		return 0, status.Errorf(codes.Internal, "Guest OS Family not supported: %v", matchedFamily)
	}
}

func (guestManager *GuestManager) GetVMLogContents(ctx context.Context, vmLogPath string) (output string, err error) {
	transferInfo, err := guestManager.FileManager.InitiateFileTransferFromGuest(ctx, guestManager.Auth, vmLogPath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error retrieving execution log from guest, %v", err))
	}

	filePath, err := receiveFileFromVM(transferInfo.Url)
	if err != nil {
		return "", err
	}

	logContentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error opening log file, %v", err))
	}
	logContent := string(logContentBytes)
	logContent = strings.TrimSuffix(logContent, "\n")
	logContent = strings.TrimSuffix(logContent, "\r")

	if err = guestManager.FileManager.DeleteFile(ctx, guestManager.Auth, vmLogPath); err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error deleting log file: %v", err))
	}
	if err = os.Remove(filePath); err != nil {
		return "", err
	}
	return logContent, nil
}

func (guestManager *GuestManager) AwaitProcessCompletion(ctx context.Context, processID int64) (bool, error) {
	for {
		log.Tracef("Program running. PID: %v", processID)

		active_process, err := guestManager.ProcessManager.ListProcesses(ctx, guestManager.Auth, []int64{processID})
		if err != nil {
			return false, status.Error(codes.Internal, fmt.Sprintf("Error listing VM process' %v", err))
		}

		vmUUID := guestManager.VirtualMachine.UUID(ctx)
		exitCode := active_process[0].ExitCode
		regexFail, _ := regexp.Compile("[1-9]+")
		if regexFail.MatchString(string(exitCode)) {
			return false, status.Error(codes.Internal, fmt.Sprintf("Error: VM UUID: `%v` PID: `%v` exit code: %v", vmUUID, processID, exitCode))

		}

		processEndTime := active_process[0].EndTime
		regexSuccess, _ := regexp.Compile("[^0-9]+")
		successCheck := regexSuccess.MatchString(string(exitCode))
		if successCheck && processEndTime != nil {
			log.Tracef("VM UUID: `%v` PID: `%v` exit code: %v", vmUUID, processID, exitCode)
			break
		}
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return true, nil
}
func (guestManager *GuestManager) AwaitProcessWithTimeout(ctx context.Context, processID int64, normalizedTargetPath string) error {
	resultChannel := make(chan bool)
	errorChannel := make(chan error)

	timeout := 300 * time.Second

	go func() {
		processFinished, err := guestManager.AwaitProcessCompletion(ctx, processID)
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

func (guestManager *GuestManager) ExecutePackageAction(ctx context.Context, action string) (commandOutput string, err error) {
	var vmManagedObject mo.VirtualMachine
	guestManager.VirtualMachine.Properties(ctx, guestManager.VirtualMachine.Reference(), []string{}, &vmManagedObject)

	stdoutBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)

	cmd := &exec.Cmd{
		Path:   action,
		Stdout: stdoutBuffer,
		Stderr: stderrBuffer,
	}

	err = guestManager.Toolbox.Run(ctx, cmd)
	stdout := strings.TrimSpace(stdoutBuffer.String())
	stderr := strings.TrimSpace(stderrBuffer.String())
	commandOutput = stdout + stderr
	return
}

func (guestManager *GuestManager) getAssetSizeAndAttributes(ctx context.Context, sourcePath string, asset []string) (int64, types.BaseGuestFileAttributes, error) {

	guestOsFamily, err := guestManager.FindGuestOSFamily(ctx)
	if err != nil {
		return 0, nil, err
	}

	var filePermissions string
	if len(asset) < 3 {
		filePermissions = ""
	} else {
		filePermissions = asset[2]
	}

	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return 0, nil, err
	}

	fileAttributes, err := createFileAttributesByOsFamily(guestOsFamily, filePermissions)
	if err != nil {
		return 0, nil, err
	}
	return fileInfo.Size(), fileAttributes, nil
}

func (guestManager *GuestManager) CopyAssetsToVM(ctx context.Context, assets [][]string, packagePath string) (assetFilePaths []string, err error) {
	for _, asset := range assets {

		sourcePath := path.Join(packagePath, filepath.FromSlash(asset[0]))
		targetPath := asset[1]

		var vmManagedObject mo.VirtualMachine
		guestManager.VirtualMachine.Properties(ctx, guestManager.VirtualMachine.Reference(), []string{}, &vmManagedObject)

		fileSize, fileAttributes, err := guestManager.getAssetSizeAndAttributes(ctx, sourcePath, asset)
		if err != nil {
			return nil, err
		}
		httpPutRequest := soap.DefaultUpload
		httpPutRequest.ContentLength = fileSize

		normalizedTargetPath := normalizePackageTargetPath(sourcePath, targetPath)
		log.Debugf("Creating Directory %s", path.Dir(normalizedTargetPath))
		err = guestManager.FileManager.MakeDirectory(ctx, guestManager.Auth, path.Dir(normalizedTargetPath), true)
		if err != nil && !strings.HasSuffix(err.Error(), "already exists") {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error creating VM directories, %v", err))
		}
		file, err := os.Open(sourcePath)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error reading source file, %v", err))
		}
		log.Debugf("Uploading file %s to %s", sourcePath, normalizedTargetPath)
		err = guestManager.Toolbox.Upload(ctx, file, normalizedTargetPath, httpPutRequest, fileAttributes, true)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Error Uploading file to VM %v", err))
		}

		splitFilepath := strings.Split(normalizedTargetPath, "/")
		if len(splitFilepath) < 2 {
			splitFilepath = strings.Split(normalizedTargetPath, "\\")
		}
		filename := splitFilepath[len(splitFilepath)-1]

		log.Infof("Successfully uploaded '%v' to VM UUID '%v'", filename, guestManager.VirtualMachine.UUID(ctx))

		assetFilePaths = append(assetFilePaths, normalizedTargetPath)
	}
	return assetFilePaths, nil
}

func (guestManager *GuestManager) UploadPackageContents(ctx context.Context, source *common.Source) (ExecutorPackage, []string, error) {
	if _, err := CheckVMStatus(ctx, guestManager.VirtualMachine); err != nil {
		return ExecutorPackage{}, nil, err
	}

	packagePath, executorPackage, err := GetPackageMetadata(
		source.GetName(),
		source.GetVersion(),
	)
	if err != nil {
		return ExecutorPackage{}, nil, err
	}

	assetFilePaths, err := guestManager.CopyAssetsToVM(ctx, executorPackage.GetAssets(), packagePath)
	if err != nil {
		return ExecutorPackage{}, nil, err
	}
	if err = CleanupTempPackage(packagePath); err != nil {
		return ExecutorPackage{}, nil, status.Error(codes.Internal, fmt.Sprintf("Error during temp Feature package cleanup (%v)", err))
	}

	return executorPackage, assetFilePaths, nil
}
