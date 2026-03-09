package api_test

import (
	"bytes"
	"context"
	"errors"
	"faas-engine-go/internal/api"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockDeployer struct {
	shouldFail bool
	called     bool
	name       string
}

func (m *mockDeployer) Deploy(ctx context.Context, name string, file io.Reader, write io.Writer) error {
	m.called = true
	m.name = name
	if m.shouldFail {
		return errors.New("deploy failed")
	}
	return nil
}

func TestDeployHandler_InvalidSize(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "deploy_test.go")
	if err != nil {
		t.Fatal(err)
	}

	large := make([]byte, 51<<20)
	if _, err := part.Write(large); err != nil {
		t.Fatal(err)
	}

	if err := writer.WriteField("name", "test-fn"); err != nil {
		t.Fatalf("failed to write name field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{})
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDeployHandler_Success(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "deploy_test.go")
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 1<<20)
	if _, err := part.Write(data); err != nil {
		t.Fatalf("failed to write file part: %v", err)
	}

	if err := writer.WriteField("name", "test-fn"); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{})
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	expected := `Your function is live at: http://test-fn.localhost`
	result := strings.TrimSpace(rr.Body.String())

	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestDeployHandler_MissingFile(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("name", "test-fn"); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{})
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDeployHandler_DeployFailure_StreamedError(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.go")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := part.Write([]byte("valid small content")); err != nil {
		t.Fatalf("failed to write payload: %v", err)
	}

	if err := writer.WriteField("name", "test-fn"); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{shouldFail: true})
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	expected := "ERROR: deploy failed"
	result := rr.Body.String()

	if !strings.Contains(result, expected) {
		t.Fatalf("expected deploy error, got %s", result)
	}
}
