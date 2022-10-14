package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	"github.com/open-cyber-range/vmware-handler/grpc/common"
	"github.com/open-cyber-range/vmware-handler/grpc/feature"
	"github.com/open-cyber-range/vmware-handler/library"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type featurerServer struct {
	feature.UnimplementedFeatureServiceServer
	Client        *govmomi.Client
	Configuration *library.Configuration
}

func sendPutRequest(url string, filePath string, caCertString string) (response *http.Response, err error) {

	log.Printf("Received put request: URL: %v\nfile_path: %v", url, filePath)

	caCertBytes := []byte(caCertString)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertBytes)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			// Certificates:       []tls.Certificate{},
			// RootCAs:            caCertPool,
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: transport, Timeout: 30 * time.Second}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	formFile, err := writer.CreateFormFile("script", "echo.sh")
	if err != nil {
		log.Fatalf("Error creating form file, %v", err)
	}
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error reading file bytes, %v", err)
	}
	fileInfo, _ := file.Stat()
	log.Printf("File size: %v", fileInfo.Size())

	_, err = io.Copy(formFile, file)
	if err != nil {
		log.Fatalf("Error copying file, %v", err)

	}
	writer.Close()

	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body.Bytes()))
	if err != nil {
		log.Fatalf("Error creating HTTP request, %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err = client.Do(request)
	if err != nil {
		log.Fatalf("Error sending HTTP request: %v", err)
	}

	return
}

// func normalizeDestinationPath(sourcePath string, destinationPath string) (string, error) {

//		if strings.HasSuffix(destinationPath, "/") {
//			sourcePathSlices := strings.Split(sourcePath, "/")
//			sourceFileName := sourcePathSlices[len(sourcePathSlices)-1]
//			destinationPath = strings.Join([]string{destinationPath, sourceFileName}, "")
//		}
//		return destinationPath, nil
//	}

func (server *featurerServer) Create(ctx context.Context, featureDeployment *feature.Feature) (identifier *common.Identifier, err error) {

	// packagePath, err := library.DownloadPackage(featureDeployment.GetSource().GetName(), featureDeployment.GetSource().GetVersion())
	// if err != nil {
	// 	log.Errorf("Failed to download package (%v)", err)
	// 	return nil, status.Error(codes.NotFound, fmt.Sprintf("Failed to download package (%v)", err))
	// }
	// log.Infof("Downloaded package to: %v", packagePath)

	// packageTomlContent, err := library.GetPackageData(packagePath) //i need to take the directory map from here but it is not implemented yet
	// if err != nil {
	// 	log.Errorf("Failed to get package data, %v", err)
	// }
	// packageFeature := (*packageTomlContent)["feature"]
	// normalize folder paths
	// make sure destination folders exist / create them

	testSourceFile := "../extra/test-deputy-packages/test-feature-package/src/echo.sh"
	testDestinationTarget := "/tmp/echo.sh"

	vmwareClient := library.NewVMWareClient(server.Client, server.Configuration.TemplateFolderPath)
	virtualMachine, err := vmwareClient.GetVirtualMachineByUUID(ctx, featureDeployment.VirtualMachineId)
	if err != nil {
		log.Fatalf("Error getting VM by UUID, %v", err)
	}

	operationsManager := guest.NewOperationsManager(virtualMachine.Client(), virtualMachine.Reference())

	fileManager, err := operationsManager.FileManager(ctx)
	if err != nil {
		log.Fatalf("Error creating FileManager, %v", err)
	}

	auth := &types.NamePasswordAuthentication{
		Username: featureDeployment.User,
		Password: featureDeployment.Password,
	}

	fileInfo, err := os.Stat(testSourceFile)
	if err != nil {
		log.Fatalf("Error getting file information, %v", err)
	}

	log.Printf("destination folder: %v", testDestinationTarget)
	log.Printf("file size: %v", fileInfo.Size())
	log.Printf("auth: %v", auth)

	fileModTime := fileInfo.ModTime()
	guestFileAttributes := &types.GuestFileAttributes{
		ModificationTime: &fileModTime,
	}
	transferUrl, err := fileManager.InitiateFileTransferToGuest(ctx, auth, testDestinationTarget, guestFileAttributes, fileInfo.Size(), true)
	if err != nil {
		log.Fatalf("Error transfering file to guest, %v", err)
	}

	//vmHostSystem, err := virtualMachine.HostSystem(ctx)
	//if err != nil {
	//	log.Fatalf("Error getting VM host system, %v", err)
	//}
	// vmCert := vmHostSystem.Client().Certificate() //this returns <nil>
	//certManager, _ := vmHostSystem.ConfigManager().CertificateManager(ctx)
	//cmCerts, _ := certManager.ListCACertificates(ctx)
	//cmCert := cmCerts[0]

	response, err := sendPutRequest(transferUrl, testSourceFile, "something")
	log.Printf("Response: %v", response)

	identifier = &common.Identifier{
		Value: "somehow made it to the end of the method",
	}
	return
}

func (server *featurerServer) Delete(ctx context.Context, identifier *common.Identifier) (*emptypb.Empty, error) {
	// todo
	log.Printf("I'm about to turn around and delete %v, so you better watch out.", identifier.GetValue())
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
		*capability.Capabilities_Switch.Enum(), //TODO: change me to feature
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
