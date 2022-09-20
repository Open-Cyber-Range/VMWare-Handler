package library

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
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
