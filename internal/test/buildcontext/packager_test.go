package test

import (
	"archive/tar"
	"faas-engine-go/internal/buildcontext"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestPackageFunction_Success(t *testing.T) {

	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "hello.txt")
	content := []byte("hello world")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}

	reader, err := buildcontext.CreateTarStream(tempDir)
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
	_, err := buildcontext.CreateTarStream("./invalid/path")

	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
