package library

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
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

	vmToolsRunning, err := CheckVMStatus(ctx, virtualMachine)
	if err != nil {
		return nil, err
	} else if !vmToolsRunning {
		err = vmwareClient.AwaitVMToolsToComeOnline(ctx, vmId)
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

	guestManager := &GuestManager{
		VirtualMachine: virtualMachine,
		Auth:           auth,
		FileManager:    fileManager,
		ProcessManager: processManager,
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
	} else {
		log.Printf("Successfully uploaded file to vm")
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
	guestManager.VirtualMachine.Properties(ctx, guestManager.VirtualMachine.Reference(), []string{}, &vmProperties)

	matchedFamily, successful_match := parseOsFamily(vmProperties.Guest.GuestFamily)
	if successful_match {
		return matchedFamily, nil
	}

	return 0, status.Error(codes.Internal, "Guest OS Family not supported")
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

	logContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("Error opening log file, %v", err))
	}

	return string(logContent), nil
}

func (guestManager *GuestManager) AwaitProcessCompletion(ctx context.Context, processID int64) (bool, error) {
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

func (guestManager *GuestManager) ExecutePackageAction(ctx context.Context, action string) (vmLog string, err error) {
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
		err = guestManager.AwaitProcessWithTimeout(ctx, processID, action)
		if err != nil {
			return
		}
	}
	vmLog, err = guestManager.GetVMLogContents(ctx, vmLogPath)
	if err != nil {
		return
	}
	log.Infof("PID %v output: %v", processID, vmLog)
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

func (guestManager *GuestManager) CopyAssetsToVM(ctx context.Context, assets [][]string, packagePath string, storage Storage[ExecutorContainer], executorId string) (err error) {
	for _, asset := range assets {

		sourcePath := path.Join(packagePath, filepath.FromSlash(asset[0]))
		targetPath := asset[1]

		fileSize, fileAttributes, err := guestManager.getAssetSizeAndAttributes(ctx, sourcePath, asset)
		if err != nil {
			return err
		}

		normalizedTargetPath := normalizePackageTargetPath(sourcePath, targetPath)

		err = guestManager.FileManager.MakeDirectory(ctx, guestManager.Auth, path.Dir(normalizedTargetPath), true)
		if err != nil && !strings.HasSuffix(err.Error(), "already exists") {
			return status.Error(codes.Internal, fmt.Sprintf("Error creating VM directories, %v", err))
		}

		transferUrl, err := guestManager.FileManager.InitiateFileTransferToGuest(ctx, guestManager.Auth, normalizedTargetPath, fileAttributes, fileSize, true)
		if err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Error creating transfer URL, %v", err))
		}

		if err = sendFileToVM(transferUrl, sourcePath); err != nil {
			return err
		}

		storage.Container.FilePaths = append(storage.Container.FilePaths, normalizedTargetPath)
		if err = storage.Update(ctx, executorId); err != nil {
			return err
		}
	}
	return nil
}
