package library

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	chromahtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/iancoleman/strcase"
	"github.com/open-cyber-range/vmware-handler/grpc/capability"
	img64 "github.com/tenkoh/goldmark-img64"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/crypto/sha3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func CreateRandomString(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
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

func GetSha256Checksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha3.New256()
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

func ConvertEnvArrayToMap(environment []string) (environmentMap map[string]string) {
	environmentMap = make(map[string]string)
	for _, line := range environment {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			environmentMap[key] = value
		}
	}
	return
}

func ConvertEnvMapToArray(environmentMap map[string]string) (environment []string) {
	for key, value := range environmentMap {
		environment = append(environment, fmt.Sprintf("%s=%s", key, value))
	}
	return
}
