package library

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/iancoleman/strcase"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const vmToolsTimeoutSec int = 120
const vmToolsSleepSec int = 5
const vmToolsCheckTries int = vmToolsTimeoutSec / vmToolsSleepSec
const tmpPackagePrefix string = "deputy-package"
const mutexTimeout time.Duration = 30 * 60 * time.Second

type Mutex struct {
	Mutex *redsync.Mutex
}
type Feature struct {
	Type   string     `json:"type"`
	Action string     `json:"action,omitempty"`
	Assets [][]string `json:"assets"`
}

type Condition struct {
	Interval uint32     `json:"interval,omitempty"`
	Action   string     `json:"action,omitempty"`
	Assets   [][]string `json:"assets"`
}

type Inject struct {
	Action string     `json:"action,omitempty"`
	Assets [][]string `json:"assets"`
}

type ExecutorPackage struct {
	Feature   Feature   `json:"feature,omitempty"`
	Condition Condition `json:"condition,omitempty"`
	Inject    Inject    `json:"inject,omitempty"`
}

func (executorPackage ExecutorPackage) GetAssets() (assets [][]string) {
	switch parcel := executorPackage; {
	case len(parcel.Feature.Assets) > 0:
		return executorPackage.Feature.Assets
	case len(parcel.Condition.Assets) > 0:
		return executorPackage.Condition.Assets
	case len(parcel.Inject.Assets) > 0:
		return executorPackage.Inject.Assets
	default:
		return
	}
}

func (executorPackage ExecutorPackage) GetAction() (action string) {
	switch parcel := executorPackage; {
	case parcel.Feature.Action != "":
		return executorPackage.Feature.Action
	case parcel.Condition.Action != "":
		return executorPackage.Condition.Action
	case parcel.Inject.Action != "":
		return executorPackage.Inject.Action
	default:
		return
	}
}

func CreateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func ScaleBytesByUnit(byteSize uint64) int32 {
	return int32(byteSize >> 20)
}

func IOReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := os.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files, nil
}

func CreateCapabilityClient(t *testing.T, serverPath string) capability.CapabilityClient {
	connection, connectionError := grpc.Dial(serverPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connectionError != nil {
		t.Fatalf("did not connect: %v", connectionError)
	}
	t.Cleanup(func() {
		connectionError := connection.Close()
		if connectionError != nil {
			t.Fatalf("Failed to close connection: %v", connectionError)
		}
	})
	return capability.NewCapabilityClient(connection)
}

func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:max]
}

const maxNameLength int = 80

func SanitizeToCompatibleName(input string) string {
	return truncateText(strcase.ToLowerCamel(input), maxNameLength)
}

func createRandomPackagePath() (string, error) {
	return os.MkdirTemp("/tmp", tmpPackagePrefix)
}

func DownloadPackage(name string, version string) (packagePath string, err error) {
	packageBasePath, err := createRandomPackagePath()

	if err != nil {
		err = fmt.Errorf("error creating tempDir for DownloadPackage: %v", err)
		return
	}

	log.Infof("Fetching package: %v, version: %v to %v", name, version, packagePath)
	downloadCommand := exec.Command("deputy", "fetch", name, "-v", version, "-s", packagePath)
	downloadCommand.Dir = packageBasePath
	output, err := downloadCommand.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v (%v)", string(output), err)
	}
	directories, err := IOReadDir(packageBasePath)
	if err != nil {
		return
	}

	if len(directories) != 1 {
		err = fmt.Errorf("expected one directory in package base path, got %v", len(directories))
		return
	}

	packageDirectory := path.Join(packageBasePath, directories[0])
	log.Infof("Downloaded package to: %v", packageDirectory)

	return packageDirectory, nil
}

func CleanupTempPackage(packagePath string) (err error) {
	packageBasePath := filepath.Clean(filepath.Join(packagePath, ".."))
	if !strings.Contains(packageBasePath, tmpPackagePrefix) {
		log.Warnf("Temp Package folder %v was not cleaned up, folder did not contain prefix %v", packageBasePath, tmpPackagePrefix)
		return nil
	}
	if err := os.RemoveAll(packageBasePath); err != nil {
		return fmt.Errorf("error deleting temp package folder %v", err)
	}

	log.Infof("Deleted temp folder %v", packageBasePath)
	return nil
}

func GetPackageChecksum(name string, version string) (checksum string, err error) {
	checksumCommand := exec.Command("deputy", "checksum", name, "-v", version)
	output, err := checksumCommand.Output()
	if err != nil {
		return
	}
	checksum = strings.TrimSpace(string(output))
	return
}

func NormalizePackageVersion(packageName string, versionRequirement string) (normalizedVersion string, err error) {
	versionCommand := exec.Command("deputy", "normalize-version", packageName, "-v", versionRequirement)
	output, err := versionCommand.Output()
	if err != nil {
		return
	}
	normalizedVersion = strings.TrimSpace(string(output))
	return
}

func GetPackageData(packagePath string) (packageData map[string]interface{}, err error) {
	packageTomlPath := path.Join(packagePath, "package.toml")
	checksumCommand := exec.Command("deputy", "parse-toml", packageTomlPath)
	output, err := checksumCommand.Output()
	if err != nil {
		return
	}
	json.Unmarshal(output, &packageData)
	return
}

func PublishTestPackage(packageFolderName string) (err error) {
	uploadCommand := exec.Command("deputy", "publish")
	workingDirectory, err := os.Getwd()
	if err != nil {
		return
	}
	uploadCommand.Dir = path.Join(workingDirectory, "..", "extra", "test-deputy-packages", packageFolderName)

	log.Infof("Publishing test package: %v", uploadCommand.Dir)

	output, err := uploadCommand.CombinedOutput()
	outputString := string(output)

	log.Infof("Publish output: `%v` ", outputString)
	if strings.Contains(outputString, "Package version on the server is either same or later") {
		return nil
	} else if err != nil {
		return fmt.Errorf("%v (%v)", outputString, err)
	}
	return
}

func CheckVMStatus(ctx context.Context, virtualMachine *object.VirtualMachine) (bool, error) {
	var vmProperties mo.VirtualMachine
	virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &vmProperties)

	vmPowerState := vmProperties.Runtime.PowerState
	vmToolsStatus := vmProperties.Guest.ToolsStatus

	if vmPowerState == types.VirtualMachinePowerStatePoweredOn {
		if vmToolsStatus == types.VirtualMachineToolsStatusToolsOk {
			return true, nil
		} else if vmToolsStatus == types.VirtualMachineToolsStatusToolsNotRunning {
			return false, nil
		}
	}

	return false, status.Error(codes.Internal, fmt.Sprintf("Error: VM Power state: %v, VM Tools status: %v", vmPowerState, vmToolsStatus))
}

func AwaitVMToolsToComeOnline(ctx context.Context, virtualMachine *object.VirtualMachine) error {
	var tries = int(vmToolsCheckTries)
	vmId := virtualMachine.UUID(ctx)

	for tries > 0 {
		tries -= 1

		var vmProperties mo.VirtualMachine
		virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &vmProperties)

		toolsStatus := vmProperties.Guest.ToolsStatus
		guestHeartBeatStatus := vmProperties.GuestHeartbeatStatus

		log.Infof("Awaiting VMTools on %v. VmTools: %v, GuestHeartBeat: %v", vmId, toolsStatus, guestHeartBeatStatus)

		if toolsStatus == types.VirtualMachineToolsStatusToolsOk &&
			guestHeartBeatStatus == types.ManagedEntityStatusGreen {
			return nil
		}

		time.Sleep(time.Second * time.Duration(vmToolsSleepSec))
	}

	return status.Error(codes.Internal, fmt.Sprintf("Timeout (%v sec) waiting for VMTools to come online on %v", vmToolsTimeoutSec, vmId))
}

func GetPackageTomlContents(packageName string, packageVersion string) (packagePath string, packageTomlContent map[string]interface{}, err error) {
	packagePath, err = DownloadPackage(packageName, packageVersion)
	if err != nil {
		return "", nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}
	packageTomlContent, err = GetPackageData(packagePath)
	if err != nil {
		log.Errorf("Failed to get package data, %v", err)
	}
	return
}

func GetPackageMetadata(packageName string, packageVersion string) (packagePath string, executorPackage ExecutorPackage, err error) {
	packagePath, packageTomlContent, err := GetPackageTomlContents(packageName, packageVersion)
	if err != nil {
		return "", ExecutorPackage{}, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package: %v", err))
	}
	infoJson, err := json.Marshal(&packageTomlContent)
	if err != nil {
		return "", ExecutorPackage{}, status.Error(codes.Internal, fmt.Sprintf("Error marshalling Toml contents: %v", err))
	}
	if err = json.Unmarshal(infoJson, &executorPackage); err != nil {
		return "", ExecutorPackage{}, status.Error(codes.Internal, fmt.Sprintf("Error unmarshalling Toml contents: %v", err))
	}
	return
}

func (mutex *Mutex) Lock() (err error) {

	successChannel := make(chan bool)
	errorChannel := make(chan error)

	go func() {
		err = mutex.Mutex.Lock()
		if err != nil {
			if strings.HasPrefix(err.Error(), "lock already taken") {
				log.Tracef("Mutex lock taken, trying again")
				time.Sleep(time.Millisecond * (time.Duration(rand.Intn(100))))

				if err = mutex.Lock(); err != nil {
					successChannel <- false
					errorChannel <- err
				}
			} else {
				successChannel <- false
				errorChannel <- err
			}
		}
		successChannel <- true
		errorChannel <- nil
	}()
	select {

	case isSuccess := <-successChannel:
		if !isSuccess {
			err := <-errorChannel
			if err != nil {
				return err
			}
		}

	case <-time.After(mutexTimeout):
		return status.Error(codes.Internal, fmt.Sprintf("Mutex lock timed out after %v seconds: %v", mutexTimeout, err))
	}

	return nil
}

func (mutex *Mutex) Unlock() (err error) {
	if isUnlocked, err := mutex.Mutex.Unlock(); !isUnlocked || err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Mutex unlock failed: %v", err))
	}
	return
}
