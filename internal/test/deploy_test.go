package test

import (
	"bytes"
	"faas-engine-go/internal/api"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeployHandler_InvalidSize(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "deploy_test.go")
	if err != nil {
		t.Fatal(err)
	}

	large := make([]byte, 51<<20)
	part.Write(large)

	writer.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/deploy",
		body,
	)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	api.DeployHandler(rr, req)

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

	large := make([]byte, 20<<20)
	part.Write(large)

	writer.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/deploy",
		body,
	)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	api.DeployHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	expected := `{"message":"File received successfully"}`
	result := strings.TrimSpace(rr.Body.String())

	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}

}

func TestDeployHandler_MissingFile(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/deploy", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	api.DeployHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
