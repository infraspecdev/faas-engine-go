package test

import (
	"bytes"
	"context"
	"errors"
	"faas-engine-go/internal/api"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type mockInvoker struct {
	shouldFail bool
}

func (m *mockInvoker) Invoke(ctx context.Context, name string, payload []byte) (any, error) {
	if m.shouldFail {
		return nil, errors.New("invoke failed")
	}
	return map[string]string{"message": "ok"}, nil
}

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

	handler := api.InvokeHandler(&mockInvoker{})
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestInvokeHandler_MissingFunctionName(t *testing.T) {

	req := httptest.NewRequest(
		http.MethodPost,
		"/functions//invoke",
		nil,
	)

	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rr := httptest.NewRecorder()

	handler := api.InvokeHandler(&mockInvoker{})
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestInvokeHandler_InternalError(t *testing.T) {

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

	handler := api.InvokeHandler(&mockInvoker{shouldFail: true})
	handler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
