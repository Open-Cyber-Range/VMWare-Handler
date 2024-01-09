package library

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-cyber-range/vmware-handler/grpc/deputy"
	log "github.com/sirupsen/logrus"
)

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

func LoginToDeputy(token string) (err error) {
	loginCommand := exec.Command("deputy", "login", "-T", token)
	output, err := loginCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v (%v)", string(output), err)
	}
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

func GetPackageChecksum(name string, version string) (checksum string, err error) {
	checksumCommand := exec.Command("deputy", "checksum", name, "-v", version)
	output, err := checksumCommand.Output()
	if err != nil {
		return
	}
	checksum = strings.TrimSpace(string(output))
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

func GetPackageTomlContents(packageName string, packageVersion string) (packagePath string, packageTomlContent map[string]interface{}, err error) {
	packagePath, err = DownloadPackage(packageName, packageVersion)
	if err != nil {
		return "", nil, err
	}
	packageTomlContent, err = GetPackageData(packagePath)
	if err != nil {
		log.Errorf("Failed to get package data, %v", err)
		return "", nil, err
	}
	return
}

func GetPackageMetadata(packageName string, packageVersion string) (packagePath string, executorPackage ExecutorPackage, err error) {
	packagePath, packageTomlContent, err := GetPackageTomlContents(packageName, packageVersion)
	if err != nil {
		return "", ExecutorPackage{}, fmt.Errorf("failed to download package: %v", err)
	}
	infoJson, err := json.Marshal(&packageTomlContent)
	if err != nil {
		return "", ExecutorPackage{}, fmt.Errorf("error marshalling Toml contents: %v", err)
	}
	if err = json.Unmarshal(infoJson, &executorPackage); err != nil {
		return "", ExecutorPackage{}, fmt.Errorf("error unmarshalling Toml contents: %v", err)
	}
	return
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
