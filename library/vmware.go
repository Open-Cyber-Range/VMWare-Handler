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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/common"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/guest/toolbox"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type VMWareClient struct {
	Client        *govmomi.Client
	templatePath  string
	configuration ConfigurationVariables
}

type GuestManager struct {
	VirtualMachine *object.VirtualMachine
	Auth           *types.NamePasswordAuthentication
	ProcessManager *guest.ProcessManager
	FileManager    *guest.FileManager
	Toolbox        *toolbox.Client
	configuration  ConfigurationVariables
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

func CheckVMStatus(ctx context.Context, virtualMachine *object.VirtualMachine) (bool, error) {
	var vmProperties mo.VirtualMachine
	virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &vmProperties)

	if vmProperties.Name == "" {
		return false, fmt.Errorf("VM does not exist, likely deleted")
	}

	vmPowerState := vmProperties.Runtime.PowerState
	if vmPowerState == types.VirtualMachinePowerStatePoweredOff {
		return false, nil
	} else {
		vmToolsStatus := vmProperties.Guest.ToolsRunningStatus
		guestHeartBeatStatus := vmProperties.GuestHeartbeatStatus

		log.Debugf("%v: %v, Heartbeat: %v", vmProperties.Config.Uuid, vmToolsStatus, guestHeartBeatStatus)
		if vmToolsStatus == string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) &&
			guestHeartBeatStatus == types.ManagedEntityStatusGreen {
			return true, nil
		} else {
			return false, nil
		}
	}
}

func AwaitVMToolsToComeOnline(ctx context.Context, virtualMachine *object.VirtualMachine, configuration ConfigurationVariables) (err error) {
	var tries = int(configuration.VmToolsTimeoutSec / configuration.VmToolsRetrySec)
	defer func() {
		if panicLog := recover(); panicLog != nil {
			log.Warnf("AwaitVMToolsToComeOnline recovered from panic: %v", panicLog)
			err = fmt.Errorf("VM was likely deleted during health check")
		}
	}()

	if virtualMachine == nil {
		log.Errorf("Virtual machine is nil, likely deleted")
		return fmt.Errorf("virtual machine is nil, likely deleted")
	}
	vmId := virtualMachine.UUID(ctx)

	for tries > 0 {
		tries -= 1
		isToolsRunning, err := CheckVMStatus(ctx, virtualMachine)
		if err != nil {
			return err
		}
		if isToolsRunning {
			return nil
		}

		time.Sleep(time.Second * time.Duration(configuration.VmToolsRetrySec))
	}

	return fmt.Errorf("timeout (%v sec) waiting for VMTools to come online on %v", configuration.VmToolsRetrySec, vmId)
}

func NewVMWareClient(ctx context.Context, client *govmomi.Client, configuration Configuration) (VMWareClient, error) {
	sessionManager := session.NewManager(client.Client)
	userSession, userSessionError := sessionManager.UserSession(ctx)
	if userSessionError != nil {
		return VMWareClient{}, userSessionError
	}

	if userSession == nil {
		client.Logout(ctx)
		hostUrl, hostError := configuration.CreateLoginURL()
		if hostError != nil {
			return VMWareClient{}, hostError
		}
		loginError := client.Login(ctx, hostUrl.User)
		if loginError != nil {
			return VMWareClient{}, loginError
		}
	}

	var mgr mo.SessionManager

	err := mo.RetrieveProperties(context.Background(), client, client.ServiceContent.PropertyCollector, *client.ServiceContent.SessionManager, &mgr)
	if err != nil {
        log.Errorf("Failed to connect to VMWare: %v", err)
        return VMWareClient{}, err
    }

	return VMWareClient{
		Client:        client,
		templatePath:  configuration.TemplateFolderPath,
		configuration: configuration.Variables,
	}, nil
}

var packageActionRetryFlag struct {
	sync.Mutex
	flag bool
}

func (vmwareClient *VMWareClient) CreateGuestManagers(ctx context.Context, vmId string, auth *types.NamePasswordAuthentication) (*GuestManager, error) {
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, vmId)
	if err != nil {
		return nil, fmt.Errorf("error getting VM by UUID, %v", err)
	}

	isVmToolsRunning, err := CheckVMStatus(ctx, virtualMachine)
	if err != nil {
		return nil, err
	} else if !isVmToolsRunning {
		err = AwaitVMToolsToComeOnline(ctx, virtualMachine, vmwareClient.configuration)
		if err != nil {
			return nil, err
		}
	}

	operationsManager := guest.NewOperationsManager(virtualMachine.Client(), virtualMachine.Reference())
	fileManager, err := operationsManager.FileManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating FileManager, %v", err)
	}

	processManager, err := operationsManager.ProcessManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating a process manager, %v", err)
	}

	var vmManagedObject mo.VirtualMachine
	virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &vmManagedObject)
	toolboxClient, err := toolbox.NewClient(ctx, processManager.Client(), vmManagedObject.Reference(), auth)
	if err != nil {
		return nil, fmt.Errorf("error creating Toolbox Client' %v", err)
	}

	guestManager := &GuestManager{
		VirtualMachine: virtualMachine,
		Auth:           auth,
		FileManager:    fileManager,
		ProcessManager: processManager,
		Toolbox:        toolboxClient,
		configuration:  vmwareClient.configuration,
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
		return fmt.Errorf("error getting VM by UUID: %v", err)
	}
	operationsManager := guest.NewOperationsManager(virtualMachine.Client(), virtualMachine.Reference())

	fileManager, err := operationsManager.FileManager(ctx)
	if err != nil {
		return fmt.Errorf("error creating FileManager: %v", err)
	}

	for _, targetFile := range executorContainer.FilePaths {
		if err = fileManager.DeleteFile(ctx, &executorContainer.Auth, string(targetFile)); err != nil {
			return fmt.Errorf("error deleting file: %v", err)
		}
		log.Infof("Deleted %v", targetFile)
	}
	return nil
}

func (vmwareClient *VMWareClient) PickHostSystem(ctx context.Context) (*object.HostSystem, error) {

	finder, _, err := vmwareClient.CreateFinderAndDatacenter()
	if err != nil {
		log.Errorf("error creating finder and datacenter")
		return nil, err
	}

	hostSystems, err := finder.HostSystemList(ctx, "*")
	if err != nil {
		log.Errorf("error getting host systems: %v", err)
		return nil, err
	}

	sort.Slice(hostSystems, func(i, j int) bool {
		return hostSystems[i].Name() < hostSystems[j].Name()
	})

	hostSystem := hostSystems[len(hostSystems)-1]

	return hostSystem, nil
}

func (vmwareClient *VMWareClient) CreateTmpNetwork(ctx context.Context, hostSystem *object.HostSystem, hostNetworkSystem *object.HostNetworkSystem) (err error) {

	hostConfigManager := object.NewHostConfigManager(vmwareClient.Client.Client, hostSystem.Reference())
	networkSystem, err := hostConfigManager.NetworkSystem(ctx)
	if err != nil {
		log.Errorf("error getting network system: %v", err)
		return err
	}

	var networkConfig mo.HostNetworkSystem
	err = property.DefaultCollector(vmwareClient.Client.Client).RetrieveOne(ctx, networkSystem.Reference(), []string{"networkConfig"}, &networkConfig)
	if err != nil {
		log.Errorf("error getting network config: %v", err)
		return err
	}

	vSwitchExists := false
	for _, vSwitch := range networkConfig.NetworkConfig.Vswitch {
		if vSwitch.Name == TmpSwitchName {
			vSwitchExists = true
			break
		}
	}

	portGroupExists := false
	for _, portGroup := range networkConfig.NetworkConfig.Portgroup {
		if portGroup.Spec.Name == TmpPortGroupName {
			portGroupExists = true
			break
		}
	}

	if !vSwitchExists {
		err = hostNetworkSystem.AddVirtualSwitch(ctx, TmpSwitchName, &types.HostVirtualSwitchSpec{
			NumPorts: 8,
		})
		if err != nil {
			log.Errorf("error creating new standard vSwitch: %v", err)
			return err
		}
	}

	if !portGroupExists {
		err = hostNetworkSystem.AddPortGroup(ctx, types.HostPortGroupSpec{
			Name:        TmpPortGroupName,
			VswitchName: TmpSwitchName,
			Policy:      types.HostNetworkPolicy{},
			VlanId:      0,
		})
		if err != nil {
			log.Errorf("error creating new port group: %v", err)
			return err
		}
	}
	return
}

func (vmwareClient *VMWareClient) CleanupTmpNetwork(ctx context.Context, hostNetworkSystem *object.HostNetworkSystem) error {
	err := hostNetworkSystem.RemovePortGroup(ctx, TmpPortGroupName)
	if err != nil {
		log.Errorf("error removing tmp port group: %v", err)
		return err
	}

	err = hostNetworkSystem.RemoveVirtualSwitch(ctx, TmpSwitchName)
	if err != nil {
		log.Errorf("error removing tmp virtual switch: %v", err)
		return err
	}
	return err
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

func receiveFileFromVM(url string) (logPath string, err error) {
	out, err := os.CreateTemp("", "executor.log")
	if err != nil {
		return "", fmt.Errorf("error creating file, %v", err)
	}

	logPath = out.Name()
	defer out.Close()

	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := http.Client{Transport: transport, Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("error sending HTTP request, %v", err)
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
		if filePermissions == "" {
			fileAttributes = &types.GuestPosixFileAttributes{}
		} else {
			permissionsInt, err := strconv.ParseInt(filePermissions, 8, 64)
			if err != nil {
				log.Errorf("Error parsing file permission str to int: %v", err)
				return nil, fmt.Errorf("error parsing file permissions: %v", err)
			}
			fileAttributes = &types.GuestPosixFileAttributes{
				Permissions: permissionsInt,
			}
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
	for tries = 0; tries < guestManager.configuration.VmPropertiesTimeoutSec; tries++ {
		err := guestManager.VirtualMachine.Properties(ctx, guestManager.VirtualMachine.Reference(), []string{}, &vmProperties)
		if err != nil {
			return 0, fmt.Errorf("error retrieving VM properties, %v", err)
		}
		if vmProperties.Guest.GuestFamily == "" || vmProperties.Guest.GuestFamily == "0" {
			log.Debugf("Awaiting VM properties to be populated for VM %v", guestManager.VirtualMachine.UUID(ctx))
			time.Sleep(1 * time.Second)
			continue

		} else {
			break
		}

	}
	vmGuestFamily := vmProperties.Guest.GuestFamily
	if tries >= guestManager.configuration.VmPropertiesTimeoutSec {
		log.Warnf("Timeout retrieving VM %v properties, defaulting to 'linuxGuest'", guestManager.VirtualMachine.UUID(ctx))
		vmGuestFamily = "linuxGuest"
	}

	matchedFamily, successful_match := parseOsFamily(vmGuestFamily)
	if successful_match {
		return matchedFamily, nil
	} else {
		return 0, fmt.Errorf("guest OS Family not supported: %v", matchedFamily)
	}
}

func (guestManager *GuestManager) GetVMLogContents(ctx context.Context, vmLogPath string) (output string, err error) {
	transferInfo, err := guestManager.FileManager.InitiateFileTransferFromGuest(ctx, guestManager.Auth, vmLogPath)
	if err != nil {
		return "", fmt.Errorf("error retrieving execution log from guest, %v", err)
	}

	filePath, err := receiveFileFromVM(transferInfo.Url)
	if err != nil {
		return "", err
	}

	logContentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening log file, %v", err)
	}
	logContent := string(logContentBytes)
	logContent = strings.TrimSuffix(logContent, "\n")
	logContent = strings.TrimSuffix(logContent, "\r")

	if err = guestManager.FileManager.DeleteFile(ctx, guestManager.Auth, vmLogPath); err != nil {
		return "", fmt.Errorf("error deleting log file: %v", err)
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
			return false, fmt.Errorf("error listing VM process' %v", err)
		}

		vmUUID := guestManager.VirtualMachine.UUID(ctx)
		exitCode := active_process[0].ExitCode
		regexFail, _ := regexp.Compile("[1-9]+")
		if regexFail.MatchString(string(exitCode)) {
			return false, fmt.Errorf("error: VM UUID: `%v` PID: `%v` exit code: %v", vmUUID, processID, exitCode)

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

func (guestManager *GuestManager) getVMEnvironment(ctx context.Context, vmManagedObject mo.VirtualMachine) (vmEnvironmentMap map[string]string, err error) {
	var printEnvCmdPath string
	if strings.Contains(strings.ToLower(vmManagedObject.Guest.GuestFullName), "windows") {
		printEnvCmdPath = "set"
	} else {
		printEnvCmdPath = "printenv"
	}

	stdoutBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)
	printEnvCmd := &exec.Cmd{
		Path:   printEnvCmdPath,
		Stdout: stdoutBuffer,
		Stderr: stderrBuffer,
	}
	if err = guestManager.Toolbox.Run(ctx, printEnvCmd); err != nil {
		log.Errorf("Error getting initial VM environment: %v", err)
		return nil, fmt.Errorf("error creating VM directories, %v", err)
	}

	vmEnvironment := strings.Split(stdoutBuffer.String(), "\n")
	for i, line := range vmEnvironment {
		vmEnvironment[i] = strings.TrimSpace(line)
	}

	vmEnvironmentMap = ConvertEnvArrayToMap(vmEnvironment)

	return
}

func isRebootRelatedError(err error) bool {
	for _, rebootError := range PotentialRebootErrorList {
		if strings.Contains(err.Error(), rebootError) {
			return true
		}
	}
	return false
}

func (guestManager *GuestManager) ExecutePackageAction(ctx context.Context, action string, environment []string) (stdout string, stderr string, err error) {
	var vmManagedObject mo.VirtualMachine
	guestManager.VirtualMachine.Properties(ctx, guestManager.VirtualMachine.Reference(), []string{}, &vmManagedObject)

	stdoutBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)

	var tries = int(guestManager.configuration.ExecutorRunTimeoutSec / guestManager.configuration.ExecutorRunRetrySec)

	for tries > 0 {
		tries -= 1

		defer func() {
			if panicLog := recover(); panicLog != nil {
				log.Warnf("Panic occurred during execution of %v, err: %v", action, panicLog)
				packageActionRetryFlag.Lock()
				defer func() {
					packageActionRetryFlag.flag = false
					defer packageActionRetryFlag.Unlock()
				}()

				if !packageActionRetryFlag.flag {
					err = AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, guestManager.configuration)
					log.Warnf("Retrying function after panic: %v", action)
					packageActionRetryFlag.flag = true
					stdout, stderr, err = guestManager.ExecutePackageAction(ctx, action, environment)
					return
				}
				log.Warnf("Repeating panic by %v, err: %v", action, panicLog)
				err = fmt.Errorf("repeating panic by %v, err: %v", action, panicLog)
			}
		}()

		vmEnvironmentMap, envErr := guestManager.getVMEnvironment(ctx, vmManagedObject)
		if envErr != nil {
			if !isRebootRelatedError(envErr) {
				logMessage := fmt.Sprintf("Error executing command:\nError: %v\nStdout: %v\nStderr: %v", envErr, stdoutBuffer.String(), stderrBuffer.String())
				log.Errorf(logMessage)
				return "", "", fmt.Errorf(logMessage)
			}

			log.Infof("Retrying action %v, err: %v", action, envErr)
			if err = AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, guestManager.configuration); err != nil {
				return "", "", err
			}
			time.Sleep(time.Duration(guestManager.configuration.ExecutorRunRetrySec) * time.Second)
			envErr = nil
			continue
		}

		inputEnvironmentMap := ConvertEnvArrayToMap(environment)
		for key, value := range inputEnvironmentMap {
			vmEnvironmentMap[key] = value
		}
		combinedEnvironment := ConvertEnvMapToArray(vmEnvironmentMap)

		stdoutBuffer = new(bytes.Buffer)
		stderrBuffer = new(bytes.Buffer)

		cmd := &exec.Cmd{
			Path:   action,
			Env:    combinedEnvironment,
			Stdout: stdoutBuffer,
			Stderr: stderrBuffer,
		}

		log.Debugf("Executing Action: %v, Input environment: %v, Final Environment: %v", action, environment, combinedEnvironment)
		runErr := guestManager.Toolbox.Run(ctx, cmd)
		if runErr != nil {
			log.Errorf("Error executing command. Error: %v. Stdout: %v. Stderr: %v", runErr, stdoutBuffer.String(), stderrBuffer.String())
			return "", "", fmt.Errorf("error executing command. Error: %v. Stdout: %v. Stderr: %v", runErr, stdoutBuffer.String(), stderrBuffer.String())
		}

		break
	}

	stdout = strings.TrimSpace(stdoutBuffer.String())
	stderr = strings.TrimSpace(stderrBuffer.String())
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
			return nil, fmt.Errorf("error creating VM directories, %v", err)
		}
		file, err := os.Open(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("error reading source file, %v", err)
		}
		log.Debugf("Uploading file %s to %s", sourcePath, normalizedTargetPath)
		err = guestManager.Toolbox.Upload(ctx, file, normalizedTargetPath, httpPutRequest, fileAttributes, true)
		if err != nil {
			return nil, fmt.Errorf("error Uploading file to VM %v", err)
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
	var tries = int(guestManager.configuration.ExecutorRunTimeoutSec / guestManager.configuration.ExecutorRunRetrySec)
	for tries > 0 {
		tries -= 1
		if err := AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, guestManager.configuration); err != nil {
			return ExecutorPackage{}, nil, err
		}

		packagePath, executorPackage, err := GetPackageMetadata(
			source.GetName(),
			source.GetVersion(),
		)
		if err != nil {
			return ExecutorPackage{}, nil, err
		}

		assetFilePaths, err := guestManager.CopyAssetsToVM(ctx, executorPackage.PackageBody.Assets, packagePath)
		if err != nil {
			if !isRebootRelatedError(err) {
				return ExecutorPackage{}, nil, err
			}

			if err = AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, guestManager.configuration); err != nil {
				return ExecutorPackage{}, nil, err
			}
			time.Sleep(time.Duration(guestManager.configuration.ExecutorRunRetrySec) * time.Second)
			err = nil

			continue
		}

		if err = CleanupTempPackage(packagePath); err != nil {
			return ExecutorPackage{}, nil, fmt.Errorf("error during temp package cleanup (%v)", err)
		}

		return executorPackage, assetFilePaths, nil
	}
	return ExecutorPackage{}, nil, fmt.Errorf("timeout uploading package contents to VM UUID: %v", guestManager.VirtualMachine.UUID(ctx))
}

func (guestManager *GuestManager) Reboot(ctx context.Context) (err error) {
	if err = guestManager.VirtualMachine.RebootGuest(ctx); err != nil {
		log.Errorf("Error rebooting VM: %v", err)
		return err
	}

	time.Sleep(time.Second * 10)
	if err = AwaitVMToolsToComeOnline(ctx, guestManager.VirtualMachine, guestManager.configuration); err != nil {
		return err
	}
	return
}
