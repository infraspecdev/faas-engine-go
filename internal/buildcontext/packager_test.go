package buildcontext

import (
	"archive/tar"
	"bytes"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func createTempFunctionDir(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "hello.txt")

	err := os.WriteFile(filePath, []byte("hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	return tempDir
}

func TestPackageFunction_Success(t *testing.T) {

	tempDir := createTempFunctionDir(t)

	reader, err := CreateTarStream(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr := tar.NewReader(reader)

	found := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading tar: %v", err)
		}

		if header.Name == "hello.txt" {
			found = true

			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}

			if string(data) != "hello world" {
				t.Fatalf("expected 'hello world', got '%s'", string(data))
			}
		}
	}

	if !found {
		t.Fatal("file not found inside tar")
	}
}

func TestPackageFunction_InvalidPath(t *testing.T) {
	_, err := CreateTarStream("./invalid/path")

	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestCreateTarStream_IncludesDockerfile_WhenMissing(t *testing.T) {
	tempDir := t.TempDir()

	// Create a sample file
	filePath := filepath.Join(tempDir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	reader, err := CreateTarStream(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr := tar.NewReader(reader)

	foundFile := false
	foundDockerfile := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading tar: %v", err)
		}

		switch header.Name {

		case "hello.txt":
			foundFile = true
			data, _ := io.ReadAll(tr)
			if string(data) != "hello world" {
				t.Fatalf("expected 'hello world', got '%s'", string(data))
			}

		case "Dockerfile":
			foundDockerfile = true
			data, _ := io.ReadAll(tr)

			target := config.ImageRef(config.RuntimesRepo, "node", "v1")

			expected := fmt.Sprintf("FROM %s\nCOPY . /function\n", target)

			if string(data) != expected {
				t.Fatalf("unexpected Dockerfile content:\nexpected:\n%s\ngot:\n%s",
					expected, string(data))
			}
		}
	}

	if !foundFile {
		t.Fatal("hello.txt not found inside tar")
	}

	if !foundDockerfile {
		t.Fatal("Dockerfile was not injected")
	}
}

func TestCreateTarStream_DoesNotOverrideExistingDockerfile(t *testing.T) {
	tempDir := t.TempDir()

	// Create existing Dockerfile
	existingContent := "FROM alpine\nCMD echo hello\n"
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	reader, err := CreateTarStream(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr := tar.NewReader(reader)

	foundDockerfile := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading tar: %v", err)
		}

		if header.Name == "Dockerfile" {
			foundDockerfile = true

			data, _ := io.ReadAll(tr)

			if string(data) != existingContent {
				t.Fatalf("Dockerfile was overridden.\nexpected:\n%s\ngot:\n%s",
					existingContent, string(data))
			}
		}
	}

	if !foundDockerfile {
		t.Fatal("Dockerfile not found in tar")
	}
}

func TestSendTarStream_Success(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			t.Fatalf("expected POST method")
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		if r.FormValue("name") != "test-function" {
			t.Fatalf("expected function name 'test-function'")
		}

		// simulate streaming deploy output
		fmt.Fprintln(w, "[1/3] Packaging function code... Done.")
		fmt.Fprintln(w, "[2/3] Building image... Done.")
		fmt.Fprintln(w, "[3/3] Pushing image... Done.")
	}))
	defer server.Close()

	tarData := bytes.NewBufferString("dummy tar content")

	err := SendTarStream(
		tarData,
		server.URL,
		"test-function",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
