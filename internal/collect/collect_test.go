package collect_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/Fuwn/kivia/internal/collect"
)

func TestFromPathFindsIdentifiersInFile(testingContext *testing.T) {
	filePath := filepath.Join("..", "..", "testdata", "samplepkg", "sample.go")
	identifiers, err := collect.FromPath(filePath)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	if len(identifiers) == 0 {
		testingContext.Fatalf("Expected identifiers, got none.")
	}

	hasFunction := false
	hasParameter := false

	for _, identifier := range identifiers {
		if identifier.Kind == "function" && identifier.Name == "Handle" {
			hasFunction = true
		}

		if identifier.Kind == "parameter" {
			hasParameter = true
		}
	}

	if !hasFunction {
		testingContext.Fatalf("Expected to find function Handle.")
	}

	if !hasParameter {
		testingContext.Fatalf("Expected to find parameters.")
	}
}

func TestFromPathFindsIdentifiersInDirectory(testingContext *testing.T) {
	rootPath := filepath.Join("..", "..", "testdata", "samplepkg")
	identifiers, err := collect.FromPath(rootPath)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	if len(identifiers) == 0 {
		testingContext.Fatalf("Expected identifiers, got none.")
	}
}

func TestFromPathFindsIdentifiersRecursively(testingContext *testing.T) {
	rootPath := filepath.Join("..", "..", "testdata", "samplepkg", "...")
	identifiers, err := collect.FromPath(rootPath)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	if len(identifiers) == 0 {
		testingContext.Fatalf("Expected identifiers, got none.")
	}
}

func TestFromPathIncludesContext(testingContext *testing.T) {
	filePath := filepath.Join("..", "..", "testdata", "samplepkg", "sample.go")
	identifiers, err := collect.FromPath(filePath)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	for _, identifier := range identifiers {
		if identifier.File == "" {
			testingContext.Fatalf("Expected identifier to have file path.")
		}

		if identifier.Line == 0 {
			testingContext.Fatalf("Expected identifier to have line number.")
		}
	}
}

func TestFromPathIgnoresUnderscore(testingContext *testing.T) {
	filePath := filepath.Join("..", "..", "testdata", "samplepkg", "sample.go")
	identifiers, err := collect.FromPath(filePath)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	for _, identifier := range identifiers {
		if identifier.Name == "_" {
			testingContext.Fatalf("Expected underscore identifiers to be ignored.")
		}
	}
}

func TestFromPathRejectsNonGoFile(testingContext *testing.T) {
	tempDir := testingContext.TempDir()
	txtFile := filepath.Join(tempDir, "readme.txt")

	if err := os.WriteFile(txtFile, []byte("hello"), 0o644); err != nil {
		testingContext.Fatalf("Failed to write test file: %v", err)
	}

	_, err := collect.FromPath(txtFile)

	if err == nil {
		testingContext.Fatalf("Expected FromPath to fail for non-Go file.")
	}
}

func TestFromPathSkipsGitVendor(testingContext *testing.T) {
	tempDir := testingContext.TempDir()

	for _, dir := range []string{".git", "vendor", "node_modules"} {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0o755); err != nil {
			testingContext.Fatalf("Failed to create directory: %v", err)
		}
	}

	goFile := filepath.Join(tempDir, "main.go")

	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		testingContext.Fatalf("Failed to write test file: %v", err)
	}

	identifiers, err := collect.FromPath(tempDir)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	if len(identifiers) != 0 {
		testingContext.Fatalf("Expected no identifiers from skipped directories.")
	}
}

func TestFromPathReturnsSortedFiles(testingContext *testing.T) {
	tempDir := testingContext.TempDir()

	for _, name := range []string{"z.go", "a.go", "m.go"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("package main\n"), 0o644); err != nil {
			testingContext.Fatalf("Failed to write test file: %v", err)
		}
	}

	identifiers, err := collect.FromPath(tempDir)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	if len(identifiers) > 0 {
		files := make(map[string]struct{})

		for _, identifier := range identifiers {
			files[identifier.File] = struct{}{}
		}

		var fileList []string

		for file := range files {
			fileList = append(fileList, filepath.Base(file))
		}

		if !isSorted(fileList) {
			testingContext.Fatalf("Expected files to be processed in sorted order.")
		}
	}
}

func TestFromPathPreservesEnclosingFunction(testingContext *testing.T) {
	filePath := filepath.Join("..", "..", "testdata", "samplepkg", "sample.go")
	identifiers, err := collect.FromPath(filePath)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	for _, identifier := range identifiers {
		if identifier.Kind == "parameter" || identifier.Kind == "receiver" {
			if identifier.Context.EnclosingFunction == "" {
				testingContext.Fatalf("Expected %s to have enclosing function.", identifier.Name)
			}
		}
	}
}

func TestFromPathIncludesTypeInformation(testingContext *testing.T) {
	filePath := filepath.Join("..", "..", "testdata", "samplepkg", "sample.go")
	identifiers, err := collect.FromPath(filePath)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	hasType := false

	for _, identifier := range identifiers {
		if identifier.Context.Type != "" {
			hasType = true

			break
		}
	}

	if !hasType {
		testingContext.Fatalf("Expected some identifiers to have type information.")
	}
}

func TestFromPathHandlesParseErrors(testingContext *testing.T) {
	tempDir := testingContext.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.go")
	validFile := filepath.Join(tempDir, "valid.go")

	if err := os.WriteFile(invalidFile, []byte("this is not valid go"), 0o644); err != nil {
		testingContext.Fatalf("Failed to write test file: %v", err)
	}

	if err := os.WriteFile(validFile, []byte("package main\nfunc main() {}"), 0o644); err != nil {
		testingContext.Fatalf("Failed to write test file: %v", err)
	}

	_, err := collect.FromPath(tempDir)

	if err == nil {
		testingContext.Fatalf("Expected FromPath to fail when parsing invalid file.")
	}
}

func TestFromPathIgnoresVendorDirectory(testingContext *testing.T) {
	tempDir := testingContext.TempDir()
	vendorDir := filepath.Join(tempDir, "vendor")

	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		testingContext.Fatalf("Failed to create vendor directory: %v", err)
	}

	vendorFile := filepath.Join(vendorDir, "vendor.go")

	if err := os.WriteFile(vendorFile, []byte("package vendor\nfunc VendorFunc() {}"), 0o644); err != nil {
		testingContext.Fatalf("Failed to write vendor file: %v", err)
	}

	mainFile := filepath.Join(tempDir, "main.go")

	if err := os.WriteFile(mainFile, []byte("package main\nfunc main() {}"), 0o644); err != nil {
		testingContext.Fatalf("Failed to write main file: %v", err)
	}

	identifiers, err := collect.FromPath(tempDir)

	if err != nil {
		testingContext.Fatalf("FromPath returned an error: %v", err)
	}

	for _, identifier := range identifiers {
		if strings.Contains(identifier.File, "vendor") {
			testingContext.Fatalf("Expected vendor directory to be skipped, found %s", identifier.File)
		}
	}
}

func isSorted(list []string) bool {
	for i := 1; i < len(list); i++ {
		if list[i-1] > list[i] {
			return false
		}
	}

	return true
}
