package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"faas-engine-go/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

type mockRoller struct {
	rollbackErr            error
	rollbackResult         *service.RollbackResult
	getHistoryErr          error
	getHistoryResult       []map[string]interface{}
	rollbackFunctionName   string
	rollbackTargetVersion  string
	getHistoryFunctionName string
	getHistoryLimit        int
}

func (m *mockRoller) Rollback(ctx context.Context, functionName, targetVersion string) (*service.RollbackResult, error) {
	m.rollbackFunctionName = functionName
	m.rollbackTargetVersion = targetVersion
	return m.rollbackResult, m.rollbackErr
}

func (m *mockRoller) GetRollbackHistory(functionName string, limit int) ([]map[string]interface{}, error) {
	m.getHistoryFunctionName = functionName
	m.getHistoryLimit = limit
	return m.getHistoryResult, m.getHistoryErr
}

func newRollbackRequest(method, url string, functionName string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	vars := map[string]string{
		"functionName": functionName,
	}
	return mux.SetURLVars(req, vars)
}

func TestRollbackHandler_InvalidFunctionName(t *testing.T) {
	req := newRollbackRequest(http.MethodPost, "/functions//rollback", "")
	rr := httptest.NewRecorder()
	handler := RollbackHandler(&mockRoller{})
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRollbackHandler_InvalidRequestBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/functions/myfunction/rollback", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Length", "12")
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHandler(&mockRoller{})
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRollbackHandler_FunctionNotFound(t *testing.T) {
	mockRoller := &mockRoller{
		rollbackErr: errors.New("function not found"),
	}

	body := map[string]string{"target_version": "v1"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/functions/myfunction/rollback", bytes.NewReader(jsonBody))
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	if mockRoller.rollbackFunctionName != "myfunction" {
		t.Fatalf("expected function name 'myfunction', got '%s'", mockRoller.rollbackFunctionName)
	}
}

func TestRollbackHandler_RollbackSuccess(t *testing.T) {
	now := time.Now()
	mockRoller := &mockRoller{
		rollbackResult: &service.RollbackResult{
			FunctionName:    "myfunction",
			PreviousVersion: "v1",
			CurrentVersion:  "v2",
			RolledBackAt:    now,
		},
	}

	body := map[string]string{"target_version": "v2"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/functions/myfunction/rollback", bytes.NewReader(jsonBody))
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result service.RollbackResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.FunctionName != "myfunction" {
		t.Fatalf("expected function name 'myfunction', got '%s'", result.FunctionName)
	}

	if result.PreviousVersion != "v1" {
		t.Fatalf("expected previous version 'v1', got '%s'", result.PreviousVersion)
	}

	if result.CurrentVersion != "v2" {
		t.Fatalf("expected current version 'v2', got '%s'", result.CurrentVersion)
	}

	if mockRoller.rollbackTargetVersion != "v2" {
		t.Fatalf("expected target version 'v2', got '%s'", mockRoller.rollbackTargetVersion)
	}
}

func TestRollbackHandler_WithoutTargetVersion(t *testing.T) {
	now := time.Now()
	mockRoller := &mockRoller{
		rollbackResult: &service.RollbackResult{
			FunctionName:    "myfunction",
			PreviousVersion: "v2",
			CurrentVersion:  "v1",
			RolledBackAt:    now,
		},
	}

	body := map[string]string{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/functions/myfunction/rollback", bytes.NewReader(jsonBody))
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// When target version is not specified, empty string should be passed
	if mockRoller.rollbackTargetVersion != "" {
		t.Fatalf("expected empty target version, got '%s'", mockRoller.rollbackTargetVersion)
	}
}

func TestRollbackHandler_ServerError(t *testing.T) {
	mockRoller := &mockRoller{
		rollbackErr: errors.New("database error"),
	}

	req := httptest.NewRequest(http.MethodPost, "/functions/myfunction/rollback", bytes.NewReader([]byte("{}")))
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestRollbackHistoryHandler_InvalidFunctionName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/functions//history", nil)
	vars := map[string]string{"functionName": ""}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHistoryHandler(&mockRoller{})
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRollbackHistoryHandler_FunctionNotFound(t *testing.T) {
	mockRoller := &mockRoller{
		getHistoryErr: errors.New("function not found"),
	}

	req := httptest.NewRequest(http.MethodGet, "/functions/myfunction/history", nil)
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHistoryHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestRollbackHistoryHandler_Success(t *testing.T) {
	history := []map[string]interface{}{
		{
			"from": "v2",
			"to":   "v1",
			"at":   time.Now().Add(-5 * time.Minute),
		},
		{
			"from": "v3",
			"to":   "v2",
			"at":   time.Now().Add(-10 * time.Minute),
		},
	}

	mockRoller := &mockRoller{
		getHistoryResult: history,
	}

	req := httptest.NewRequest(http.MethodGet, "/functions/myfunction/history?limit=20", nil)
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHistoryHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(result))
	}

	if mockRoller.getHistoryFunctionName != "myfunction" {
		t.Fatalf("expected function name 'myfunction', got '%s'", mockRoller.getHistoryFunctionName)
	}

	if mockRoller.getHistoryLimit != 20 {
		t.Fatalf("expected limit 20, got %d", mockRoller.getHistoryLimit)
	}
}

func TestRollbackHistoryHandler_DefaultLimit(t *testing.T) {
	history := []map[string]interface{}{
		{
			"from": "v2",
			"to":   "v1",
			"at":   time.Now(),
		},
	}

	mockRoller := &mockRoller{
		getHistoryResult: history,
	}

	req := httptest.NewRequest(http.MethodGet, "/functions/myfunction/history", nil)
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHistoryHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if mockRoller.getHistoryLimit != 20 {
		t.Fatalf("expected default limit 20, got %d", mockRoller.getHistoryLimit)
	}
}

func TestRollbackHistoryHandler_CustomLimit(t *testing.T) {
	history := []map[string]interface{}{}

	mockRoller := &mockRoller{
		getHistoryResult: history,
	}

	req := httptest.NewRequest(http.MethodGet, "/functions/myfunction/history?limit=50", nil)
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHistoryHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if mockRoller.getHistoryLimit != 50 {
		t.Fatalf("expected limit 50, got %d", mockRoller.getHistoryLimit)
	}
}

func TestRollbackHistoryHandler_InvalidLimit(t *testing.T) {
	history := []map[string]interface{}{}

	mockRoller := &mockRoller{
		getHistoryResult: history,
	}

	req := httptest.NewRequest(http.MethodGet, "/functions/myfunction/history?limit=invalid", nil)
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHistoryHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Should default to 20 when invalid limit provided
	if mockRoller.getHistoryLimit != 20 {
		t.Fatalf("expected default limit 20 for invalid input, got %d", mockRoller.getHistoryLimit)
	}
}

func TestRollbackHistoryHandler_EmptyHistory(t *testing.T) {
	mockRoller := &mockRoller{
		getHistoryResult: []map[string]interface{}{},
	}

	req := httptest.NewRequest(http.MethodGet, "/functions/myfunction/history", nil)
	vars := map[string]string{"functionName": "myfunction"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := RollbackHistoryHandler(mockRoller)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var result []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty history, got %d items", len(result))
	}
}
