package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"faas-engine-go/internal/api"
	"faas-engine-go/internal/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type MockScheduler struct {
	AddFn           func(string, string, []byte) error
	DeleteFn        func(string) error
	ListFn          func() []types.Schedule
	GetByFunctionFn func(string) []types.Schedule
}

func (m *MockScheduler) AddSchedule(fn, cron string, payload []byte) error {
	if m.AddFn != nil {
		return m.AddFn(fn, cron, payload)
	}
	return nil
}

func (m *MockScheduler) DeleteSchedule(id string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}
	return nil
}

func (m *MockScheduler) ListSchedules() []types.Schedule {
	if m.ListFn != nil {
		return m.ListFn()
	}
	return nil
}

func (m *MockScheduler) GetSchedulesByFunction(fn string) []types.Schedule {
	if m.GetByFunctionFn != nil {
		return m.GetByFunctionFn(fn)
	}
	return nil
}

func TestScheduleHandler_Success(t *testing.T) {
	mock := &MockScheduler{
		AddFn: func(fn, cron string, payload []byte) error {
			if fn != "testFunc" {
				t.Errorf("expected functionName testFunc, got %s", fn)
			}
			if cron != "*/1 * * * *" {
				t.Errorf("unexpected cron: %s", cron)
			}
			return nil
		},
	}

	handler := api.ScheduleHandler(mock)

	body := map[string]interface{}{
		"cron":    "*/1 * * * *",
		"payload": map[string]int{"a": 1},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/schedule/testFunc", bytes.NewBuffer(jsonBody))
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "testFunc",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}
}

func TestScheduleHandler_InvalidJSON(t *testing.T) {
	mock := &MockScheduler{}
	handler := api.ScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/schedule/testFunc", bytes.NewBuffer([]byte(`invalid-json`)))
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "testFunc",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestScheduleHandler_MissingCron(t *testing.T) {
	mock := &MockScheduler{}
	handler := api.ScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/schedule/testFunc", bytes.NewBuffer([]byte(`{}`)))
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "testFunc",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestScheduleHandler_MissingFunctionName(t *testing.T) {
	mock := &MockScheduler{}
	handler := api.ScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/schedule/", bytes.NewBuffer([]byte(`{"cron":"* * * * *"}`)))
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestDeleteScheduleHandler_Success(t *testing.T) {
	mock := &MockScheduler{
		DeleteFn: func(id string) error {
			return nil
		},
	}

	handler := api.DeleteScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/schedule/123", nil)
	req = mux.SetURLVars(req, map[string]string{
		"scheduleID": "123",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestDeleteScheduleHandler_NotFound(t *testing.T) {
	mock := &MockScheduler{
		DeleteFn: func(id string) error {
			return errors.New("not found")
		},
	}

	handler := api.DeleteScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/schedule/123", nil)
	req = mux.SetURLVars(req, map[string]string{
		"scheduleID": "123",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteScheduleHandler_MissingID(t *testing.T) {
	mock := &MockScheduler{}
	handler := api.DeleteScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/schedule/", nil)
	req = mux.SetURLVars(req, map[string]string{
		"scheduleID": "",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestListSchedulesHandler(t *testing.T) {
	mock := &MockScheduler{
		ListFn: func() []types.Schedule {
			return []types.Schedule{
				{ID: "1", FunctionName: "func1"},
			}
		},
	}

	handler := api.ListSchedulesHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/schedules", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp []types.Schedule
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("invalid JSON response")
	}

	if len(resp) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(resp))
	}
}

func TestGetSchedulesByFunctionHandler_Success(t *testing.T) {
	mock := &MockScheduler{
		GetByFunctionFn: func(fn string) []types.Schedule {
			return []types.Schedule{
				{ID: "1", FunctionName: fn},
			}
		},
	}

	handler := api.GetSchedulesByFunctionHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/schedule/testFunc", nil)
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "testFunc",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp []types.Schedule
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("invalid JSON response")
	}

	if len(resp) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(resp))
	}
}

func TestGetSchedulesByFunctionHandler_MissingFunctionName(t *testing.T) {
	mock := &MockScheduler{}
	handler := api.GetSchedulesByFunctionHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/schedule/", nil)
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
