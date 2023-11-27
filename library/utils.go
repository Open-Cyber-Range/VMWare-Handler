package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	chromahtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/iancoleman/strcase"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/deputy"
	log "github.com/sirupsen/logrus"
	img64 "github.com/tenkoh/goldmark-img64"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type MutexPool struct {
	Id            string
	Redsync       *redsync.Redsync
	RedisClient   *redis.Client
	Configuration ConfigurationVariables
}
type Mutex struct {
	PoolId           string
	Mutex            *redsync.Mutex
	RedisClient      *redis.Client
	RetryIntervalMax int
	RetryIntervalMin int
	Timeout          int
}
type Feature struct {
	Type     string `json:"type"`
	Action   string `json:"action,omitempty"`
	Restarts bool   `json:"restarts,omitempty"`
}

type Condition struct {
	Interval uint32 `json:"interval,omitempty"`
	Action   string `json:"action,omitempty"`
}

type Inject struct {
	Action   string `json:"action,omitempty"`
	Restarts bool   `json:"restarts,omitempty"`
}

type Event struct {
	FilePath string `json:"file_path"`
}

type Exercise struct {
	FilePath string `json:"file_path"`
}

type PackageBody struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Authors     []string   `json:"authors,omitempty"`
	Version     string     `json:"version"`
	License     string     `json:"license"`
	Assets      [][]string `json:"assets,omitempty"`
}

type ExecutorPackage struct {
	PackageBody PackageBody `json:"package"`
	Feature     Feature     `json:"feature,omitempty"`
	Condition   Condition   `json:"condition,omitempty"`
	Inject      Inject      `json:"inject,omitempty"`
	Event       Event       `json:"event,omitempty"`
	Exercise    Exercise    `json:"exercise,omitempty"`
}

func (mutexPool MutexPool) GetMutex(ctx context.Context, optionalId ...string) (mutex *Mutex, err error) {
	identifier := "general-mutex"
	if len(optionalId) != 0 {
		identifier = "vm-mutex-" + optionalId[len(optionalId)-1]
	}

	currentConnections, err := mutexPool.RedisClient.HLen(ctx, mutexPool.Id).Result()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting Redis entry, %v", err))
	}
	lockAlreadyExists, err := mutexPool.RedisClient.HExists(ctx, mutexPool.Id, identifier).Result()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error checking existence of Redis entry, %v", err))
	}
	if currentConnections >= mutexPool.Configuration.MaxConnections || lockAlreadyExists {
		time.Sleep(time.Duration(rand.Intn(mutexPool.Configuration.MutexPoolMaxRetryMillis)+mutexPool.Configuration.MutexPoolMinRetryMillis) * time.Millisecond)
		return mutexPool.GetMutex(ctx, optionalId...)
	}
	_, err = mutexPool.RedisClient.HSet(ctx, mutexPool.Id, identifier, 0).Result()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error setting Redis entry, %v", err))
	}

	return &Mutex{
		PoolId:      mutexPool.Id,
		Mutex:       mutexPool.Redsync.NewMutex(identifier),
		RedisClient: mutexPool.RedisClient,
		Timeout:     mutexPool.Configuration.MutexTimeoutSec,
	}, nil
}

func (mutex *Mutex) Lock(ctx context.Context) (err error) {
	successChannel := make(chan bool)
	errorChannel := make(chan error)

	go func() {
		err = mutex.Mutex.Lock()
		if err != nil {
			if strings.HasPrefix(err.Error(), "lock already taken") {
				log.Tracef("Mutex lock taken, trying again")
				time.Sleep(time.Millisecond * (time.Duration(rand.Intn(100) + 50)))

				if err = mutex.Lock(ctx); err != nil {
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

	case <-time.After(time.Duration(mutex.Timeout) * time.Second):
		return status.Error(codes.Internal, fmt.Sprintf("Mutex lock timed out after %v seconds: %v", mutex.Timeout, err))
	}

	return nil
}

func (mutex *Mutex) Unlock(ctx context.Context) (err error) {
	mutex.Mutex.Unlock()
	_, err = mutex.RedisClient.HDel(ctx, mutex.PoolId, mutex.Mutex.Name()).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error deleting Redis entry, %v", err))
	}
	return
}

func NewMutexPool(ctx context.Context, poolIdentifier string, redsync redsync.Redsync, redisClient redis.Client, configuration ConfigurationVariables) (mutexPool MutexPool, err error) {
	mutexPool = MutexPool{
		Id:            poolIdentifier,
		Redsync:       &redsync,
		RedisClient:   &redisClient,
		Configuration: configuration,
	}
	_, err = redisClient.Del(ctx, poolIdentifier).Result()
	if err != nil {
		return MutexPool{}, status.Error(codes.Internal, fmt.Sprintf("Error cleaning up MutexPool, %v", err))
	}
	return mutexPool, nil
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
	return os.MkdirTemp("/tmp", TmpPackagePrefix+"*")
}

func DownloadPackage(name string, version string) (packagePath string, err error) {
	packageBasePath, err := createRandomPackagePath()

	if err != nil {
		err = fmt.Errorf("error creating tempDir for DownloadPackage: %v", err)
		return
	}

	log.Infof("Fetching package: %v, version: %v to %v", name, version, packageBasePath)
	downloadCommand := exec.Command("deputy", "fetch", name, "-v", version, "-s", packageBasePath)
	downloadCommand.Dir = packageBasePath
	output, err := downloadCommand.CombinedOutput()
	if err != nil {
		log.Errorf("Deputy fetch command failed, %v", err)
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
	if !strings.Contains(packageBasePath, TmpPackagePrefix) {
		log.Warnf("Temp Package folder %v was not cleaned up, folder did not contain prefix %v", packageBasePath, TmpPackagePrefix)
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
	inspectCommand := exec.Command("deputy", "inspect", "-p", packagePath)
	output, err := inspectCommand.Output()
	if err != nil {
		log.Errorf("Deputy inspect command failed, %v", err)
		return
	}
	if err = json.Unmarshal(output, &packageData); err != nil {
		return nil, fmt.Errorf("error unmarshalling package data, %v", err)
	}

	if len(packageData) == 0 {
		return nil, fmt.Errorf("unexpected result (is empty)")
	}

	return
}

func LoginToDeputy(token string) (err error) {
	loginCommand := exec.Command("deputy", "login", "-T", token)
	output, err := loginCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v (%v)", string(output), err)
	}
	return
}

func PublishTestPackage(packageFolderName string, token string) (err error) {
	if err = LoginToDeputy(token); err != nil {
		return
	}

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
	if strings.Contains(outputString, "already exists") {
		return nil
	} else if err != nil {
		return fmt.Errorf("%v (%v)", outputString, err)
	}
	return
}

func CheckVMStatus(ctx context.Context, virtualMachine *object.VirtualMachine) (bool, error) {
	var vmProperties mo.VirtualMachine
	virtualMachine.Properties(ctx, virtualMachine.Reference(), []string{}, &vmProperties)

	if vmProperties.Name == "" {
		return false, status.Error(codes.Internal, "VM does not exist, likely deleted")
	}

	vmPowerState := vmProperties.Runtime.PowerState
	if vmPowerState == types.VirtualMachinePowerStatePoweredOn {
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
	return false, nil
}

func AwaitVMToolsToComeOnline(ctx context.Context, virtualMachine *object.VirtualMachine, configuration ConfigurationVariables) (err error) {
	var tries = int(configuration.VmToolsTimeoutSec / configuration.VmToolsRetrySec)
	defer func() {
		if panicLog := recover(); panicLog != nil {
			log.Warnf("AwaitVMToolsToComeOnline recovered from panic: %v", panicLog)
			err = status.Error(codes.Internal, "VM was likely deleted during health check")
		}
	}()

	if virtualMachine == nil {
		log.Errorf("Virtual machine is nil, likely deleted")
		return status.Error(codes.Internal, "Virtual machine is nil, likely deleted")
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

	return status.Error(codes.Internal, fmt.Sprintf("Timeout (%v sec) waiting for VMTools to come online on %v", configuration.VmToolsRetrySec, vmId))
}

func GetPackageTomlContents(packageName string, packageVersion string) (packagePath string, packageTomlContent map[string]interface{}, err error) {
	packagePath, err = DownloadPackage(packageName, packageVersion)
	if err != nil {
		return "", nil, err
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

func GetSha256Checksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)
	return hex.EncodeToString(hashInBytes), nil
}

func buildMarkdownConverter(markdownFilePath string) goldmark.Markdown {
	parserOptions := []parser.Option{
		parser.WithAutoHeadingID(),
	}

	rendererOptions := []renderer.Option{
		img64.WithParentPath(filepath.Dir(markdownFilePath)),
		html.WithXHTML(),
		html.WithUnsafe(),
	}

	extensions := []goldmark.Extender{
		img64.Img64,
		extension.GFM,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			),
		),
	}

	return goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(parserOptions...),
		goldmark.WithRendererOptions(rendererOptions...),
	)
}

func ConvertMarkdownToHtml(markdownFilePath string) (htmlPath string, checksum string, err error) {
	filename := filepath.Base(markdownFilePath)
	htmlFolderPath, err := createRandomPackagePath()
	if err != nil {
		return "", "", fmt.Errorf("error creating tempDir for ConvertMarkdownToHtml: %v", err)
	}

	outputFilePath := htmlFolderPath + "/" + strings.TrimSuffix(filename, filepath.Ext(filename)) + ".html"

	md, err := os.ReadFile(markdownFilePath)
	if err != nil {
		return "", "", fmt.Errorf("error reading file: %v", err)
	}

	converter := buildMarkdownConverter(markdownFilePath)

	var buff bytes.Buffer
	if err := converter.Convert(md, &buff); err != nil {
		return "", "", fmt.Errorf("error converting file: %v", err)
	}

	err = os.WriteFile(outputFilePath, buff.Bytes(), 0644)
	if err != nil {
		return "", "", fmt.Errorf("error writing file: %v", err)
	}

	checksum, err = GetSha256Checksum(outputFilePath)
	if err != nil {
		return "", "", fmt.Errorf("error getting checksum: %v", err)
	}

	return outputFilePath, checksum, nil
}

func ParseListCommandOutput(commandOutput []byte) (packages []*deputy.Package, err error) {
	trimmedString := strings.TrimSuffix(string(commandOutput), "\n")
	packageStringList := strings.Split(trimmedString, "\n")
	re := regexp.MustCompile(`([^/]+)/([^,]+),\s(.*)`)

	for _, packageString := range packageStringList {
		matches := re.FindStringSubmatch(packageString)
		if len(matches) != 4 {
			return nil, fmt.Errorf("error parsing output of deputy list command, expected 4 matches per line, got %v", len(matches))
		}
		packageName, packageType, packageVersion := matches[1], matches[2], matches[3]

		if packageName == "" || packageType == "" || packageVersion == "" {
			return nil, fmt.Errorf("error parsing output of deputy list command, one of the values was empty")
		}

		packages = append(packages, &deputy.Package{
			Name:    packageName,
			Version: packageVersion,
			Type:    packageType,
		})
	}

	return
}
