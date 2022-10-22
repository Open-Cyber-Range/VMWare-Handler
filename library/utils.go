package library

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
	return ioutil.TempDir("/tmp", "deputy-package")
}

func DownloadPackage(name string, version string) (packagePath string, err error) {
	packageBasePath, err := createRandomPackagePath()

	if err != nil {
		return
	}
	log.Infof("Created package base path: %v", packageBasePath)

	downloadCommand := exec.Command("deputy", "fetch", name, "-v", version, "-s", packagePath)
	downloadCommand.Dir = packageBasePath
	_, err = downloadCommand.Output()
	if err != nil {
		return
	}
	directories, err := IOReadDir(packageBasePath)
	if err != nil {
		return
	}

	if len(directories) != 1 {
		err = fmt.Errorf("expected one directory in package base path, got %v", len(directories))
		return
	}

	return path.Join(packageBasePath, directories[0]), nil
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
	output, err := uploadCommand.CombinedOutput()
	if strings.Contains(string(output), "Package version on the server is either same or later") {
		return nil
	} else if err != nil {
		return fmt.Errorf("%v (%v)", string(output), err)
	}
	return
}
