package api_test

import (
	"bytes"
	"context"
	"errors"
	"faas-engine-go/internal/api"
	"faas-engine-go/internal/sqlite/models"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockDeployer struct {
	shouldFail bool
}

func (m *mockDeployer) Deploy(ctx context.Context, name string, file io.Reader, out io.Writer) error {
	if m.shouldFail {
		return errors.New("deploy failed")
	}
	return nil
}

type fakeStore struct {
	version string
}

func (f *fakeStore) GetNextVersion(name string) (string, error) {
	return f.version, nil
}

func (f *fakeStore) DeactivateFunctions(name string) error {
	return nil
}

func (f *fakeStore) CreateFunction(fn *models.Function) error {
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

	_ = writer.WriteField("name", "test-fn")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{}, &fakeStore{version: "v1"})
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
	_, _ = part.Write(data)

	_ = writer.WriteField("name", "test-fn")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{}, &fakeStore{version: "v1"})
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	result := rr.Body.String()

	if !strings.Contains(result, "Your function is live at: http://test-fn.localhost") {
		t.Fatalf("unexpected response: %s", result)
	}
}

func TestDeployHandler_MissingFile(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("name", "test-fn")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{}, &fakeStore{version: "v1"})
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDeployHandler_InternalError(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.go")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = part.Write([]byte("valid content"))

	_ = writer.WriteField("name", "test-fn")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/functions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := api.DeployHandler(&mockDeployer{shouldFail: true}, &fakeStore{version: "v1"})
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	result := rr.Body.String()

	if !strings.Contains(result, "ERROR: deploy failed") {
		t.Fatalf("unexpected response: %s", result)
	}
}
