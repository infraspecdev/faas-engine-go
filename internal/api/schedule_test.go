package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

// -------------------- MOCK SCHEDULER --------------------

type mockScheduler struct {
	registerCalled bool
	removeCalled   bool
	shouldFail     bool
}

func (m *mockScheduler) RegisterSchedule(s models.Schedule) error {
	m.registerCalled = true
	if m.shouldFail {
		return context.Canceled
	}
	return nil
}

func (m *mockScheduler) RemoveSchedule(id string) {
	m.removeCalled = true
}

// -------------------- TEST DB SETUP --------------------
func setupTestDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	sqlite.DB = db

	_, err = db.Exec(`
	CREATE TABLE functions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		version TEXT,
		package_checksum TEXT,
		image TEXT,
		runtime TEXT,
		schedule_cron TEXT,
		endpoint TEXT,
		status TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE schedules (
		id TEXT PRIMARY KEY,
		function_id INTEGER NOT NULL,
		cron_expr TEXT NOT NULL,
		payload TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
	INSERT INTO functions (
		id, name, version, package_checksum, image, runtime,
		schedule_cron, endpoint, status, created_at
	)
	VALUES (
		1, 'test-fn', 'v1', 'abc', 'img', 'go',
		'', '', 'active', CURRENT_TIMESTAMP
	)
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
	INSERT INTO schedules (id, function_id, cron_expr, payload)
	VALUES ('123', 1, '* * * * *', '{}')
	`)
	if err != nil {
		t.Fatal(err)
	}
}

// -------------------- HELPERS --------------------

func setupRequest(method, url string, body []byte, vars map[string]string) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewBuffer(body))
	req = mux.SetURLVars(req, vars)
	return req
}

// -------------------- CREATE TESTS --------------------

func TestCreateSchedule_Success(t *testing.T) {
	setupTestDB(t)

	scheduler := &mockScheduler{}

	body := map[string]any{
		"cron": "*/1 * * * *",
	}
	b, _ := json.Marshal(body)

	req := setupRequest("POST", "/schedules/test-fn", b, map[string]string{
		"functionName": "test-fn",
	})

	w := httptest.NewRecorder()

	CreateScheduleHandler(scheduler)(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	if !scheduler.registerCalled {
		t.Fatal("scheduler not called")
	}
}

func TestCreateSchedule_InvalidBody(t *testing.T) {
	setupTestDB(t)

	scheduler := &mockScheduler{}

	req := setupRequest("POST", "/schedules/test-fn", []byte("{invalid"), map[string]string{
		"functionName": "test-fn",
	})

	w := httptest.NewRecorder()

	CreateScheduleHandler(scheduler)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateSchedule_MissingCron(t *testing.T) {
	setupTestDB(t)

	scheduler := &mockScheduler{}

	body := map[string]any{}
	b, _ := json.Marshal(body)

	req := setupRequest("POST", "/schedules/test-fn", b, map[string]string{
		"functionName": "test-fn",
	})

	w := httptest.NewRecorder()

	CreateScheduleHandler(scheduler)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateSchedule_SchedulerFails(t *testing.T) {
	setupTestDB(t)

	scheduler := &mockScheduler{shouldFail: true}

	body := map[string]any{
		"cron": "*/1 * * * *",
	}
	b, _ := json.Marshal(body)

	req := setupRequest("POST", "/schedules/test-fn", b, map[string]string{
		"functionName": "test-fn",
	})

	w := httptest.NewRecorder()

	CreateScheduleHandler(scheduler)(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body: %s", w.Code, w.Body.String())
	}
}

// -------------------- DELETE TESTS --------------------

func TestDeleteSchedule_Success(t *testing.T) {
	setupTestDB(t)

	scheduler := &mockScheduler{}

	req := setupRequest("DELETE", "/schedules/123", nil, map[string]string{
		"id": "123",
	})

	w := httptest.NewRecorder()

	DeleteScheduleHandler(scheduler)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !scheduler.removeCalled {
		t.Fatal("scheduler.RemoveSchedule not called")
	}
}

func TestDeleteSchedule_MissingID(t *testing.T) {
	setupTestDB(t)

	scheduler := &mockScheduler{}

	req := setupRequest("DELETE", "/schedules/", nil, map[string]string{
		"id": "",
	})

	w := httptest.NewRecorder()

	DeleteScheduleHandler(scheduler)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// -------------------- LIST TESTS --------------------

func TestListSchedules(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/schedules", nil)
	w := httptest.NewRecorder()

	ListSchedulesHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestListSchedulesByFunctionName_MissingName(t *testing.T) {
	setupTestDB(t)

	req := setupRequest("GET", "/schedules/fn", nil, map[string]string{
		"functionName": "",
	})

	w := httptest.NewRecorder()

	ListScheduleByFunctionNameHandler(nil)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
