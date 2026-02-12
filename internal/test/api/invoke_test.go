package test

import (
	"bytes"
	"faas-engine-go/internal/api"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestInvokeHandler_Success(t *testing.T) {

	body := []byte(`{"cmd":"echo hello world"}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/functions/alpine/invoke",
		bytes.NewBuffer(body),
	)

	req = mux.SetURLVars(req, map[string]string{
		"functionName": "alpine",
	})

	rr := httptest.NewRecorder()

	api.InvokeHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestInvokeHandler_MissingFunctionName(t *testing.T) {
	body := []byte(`{"cmd":"echo hello world"}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/functions/functionName/invoke",
		bytes.NewBuffer(body),
	)

	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rr := httptest.NewRecorder()

	api.InvokeHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

}

func TestInvokeHandler_InvalidJSON(t *testing.T) {

	body := []byte(`{"new":"echo hello world"}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/functions/alpine/invoke",
		bytes.NewBuffer(body),
	)

	req = mux.SetURLVars(req, map[string]string{
		"functionName": "alpine",
	})

	rr := httptest.NewRecorder()

	api.InvokeHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestInvokeHandler_MissingCmd(t *testing.T) {

	body := []byte(`{"cmd":""}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/functions/alpine/invoke",
		bytes.NewBuffer(body),
	)

	req = mux.SetURLVars(req, map[string]string{
		"functionName": "alpine",
	})

	rr := httptest.NewRecorder()

	api.InvokeHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	expected := "cmd is required"
	result := strings.TrimSpace(rr.Body.String())

	if result != expected {
		t.Fatalf("expected '%s', got '%s'", expected, result)
	}
}
