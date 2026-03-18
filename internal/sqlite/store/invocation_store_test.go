package store

import (
	"database/sql"
	"encoding/json"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupInvocationDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	sqlite.DB = db

	if err := sqlite.InitTables(); err != nil {
		t.Fatal(err)
	}

	return db
}

func newInvocation(status string) *models.Invocation {
	return &models.Invocation{
		FunctionID:      1,
		ContainerID:     "c1",
		TriggerType:     "http",
		Status:          status,
		ExitCode:        0,
		DurationMs:      0,
		RequestPayload:  json.RawMessage(`{}`),
		ResponsePayload: json.RawMessage(`{}`),
		LogsPath:        "",
		StartedAt:       time.Now(),
		FinishedAt:      time.Now(),
	}
}

func TestCreateAndGetInvocation(t *testing.T) {
	db := setupInvocationDB(t)

	inv := newInvocation("pending")

	if err := CreateInvocation(db, inv); err != nil {
		t.Fatal(err)
	}

	res, err := GetInvocationByID(db, inv.ID)
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || res.ID != inv.ID {
		t.Fatalf("expected invocation, got %+v", res)
	}
}

func TestGetInvocationByID_NotFound(t *testing.T) {
	db := setupInvocationDB(t)

	res, err := GetInvocationByID(db, "unknown")
	if err != nil {
		t.Fatal(err)
	}

	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}
}

func TestMarkInvocationRunning(t *testing.T) {
	db := setupInvocationDB(t)

	inv := newInvocation("pending")
	_ = CreateInvocation(db, inv)

	if err := MarkInvocationRunning(db, inv.ID, "container-1"); err != nil {
		t.Fatal(err)
	}

	res, _ := GetInvocationByID(db, inv.ID)

	if res.Status != "running" || res.ContainerID != "container-1" {
		t.Fatalf("expected running state, got %+v", res)
	}
}

func TestCompleteInvocation(t *testing.T) {
	db := setupInvocationDB(t)

	start := time.Now().Add(-2 * time.Second)

	inv := newInvocation("pending")
	_ = CreateInvocation(db, inv)

	if err := CompleteInvocation(
		db,
		inv.ID,
		"success",
		0,
		json.RawMessage(`{"ok":true}`),
		"/logs/path",
		start,
	); err != nil {
		t.Fatal(err)
	}

	res, _ := GetInvocationByID(db, inv.ID)

	if res.Status != "success" || res.ExitCode != 0 {
		t.Fatalf("expected completed invocation, got %+v", res)
	}

	if res.DurationMs <= 0 {
		t.Fatalf("expected duration > 0")
	}
}

func TestListInvocationsByFunction(t *testing.T) {
	db := setupInvocationDB(t)

	for i := 0; i < 3; i++ {
		_ = CreateInvocation(db, newInvocation("pending"))
	}

	list, err := ListInvocationsByFunction(db, 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
}

func TestListInvocationsByStatus(t *testing.T) {
	db := setupInvocationDB(t)

	_ = CreateInvocation(db, newInvocation("success"))
	_ = CreateInvocation(db, newInvocation("failed"))

	list, err := ListInvocationsByStatus(db, "success", 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 1 || list[0].Status != "success" {
		t.Fatalf("expected 1 success invocation, got %+v", list)
	}
}
