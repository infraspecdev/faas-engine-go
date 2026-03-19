package service

import (
	"errors"
	"testing"

	"faas-engine-go/internal/sqlite/models"
)

func TestGetLogs_Success(t *testing.T) {

	logService := &LogService{
		getLogs: func(functionID int, limit int) ([]models.Invocation, error) {
			return []models.Invocation{
				{
					ID:         "inv1",
					Status:     "success",
					Logs:       "log1",
					DurationMs: 100,
				},
				{
					ID:         "inv2",
					Status:     "failed",
					Logs:       "log2",
					DurationMs: 200,
				},
			}, nil
		},
	}

	res, err := logService.GetLogs(1, 10)

	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(res))
	}

	if res[0].ID != "inv1" || res[1].ID != "inv2" {
		t.Fatal("unexpected log IDs")
	}
}

func TestGetLogs_Error(t *testing.T) {

	logService := &LogService{
		getLogs: func(functionID int, limit int) ([]models.Invocation, error) {
			return nil, errors.New("db error")
		},
	}

	_, err := logService.GetLogs(1, 10)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLogsByName_Success(t *testing.T) {

	logService := &LogService{
		getActiveFunctionID: func(name string) (int, error) {
			return 1, nil
		},
		getLogs: func(functionID int, limit int) ([]models.Invocation, error) {
			return []models.Invocation{
				{ID: "inv1", Status: "success", Logs: "ok", DurationMs: 50},
			}, nil
		},
	}

	res, err := logService.GetLogsByName("test", 10)

	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 1 {
		t.Fatal("expected 1 log")
	}
}

func TestGetLogsByName_FunctionNotFound(t *testing.T) {

	logService := &LogService{
		getActiveFunctionID: func(name string) (int, error) {
			return 0, errors.New("not found")
		},
	}

	_, err := logService.GetLogsByName("test", 10)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLogsByNameAndVersion_Success(t *testing.T) {

	logService := &LogService{
		getFunctionByVersion: func(name, version string) (int, error) {
			return 2, nil
		},
		getLogs: func(functionID int, limit int) ([]models.Invocation, error) {
			return []models.Invocation{
				{ID: "invX", Status: "success", Logs: "v2", DurationMs: 70},
			}, nil
		},
	}

	res, err := logService.GetLogsByNameAndVersion("test", "v2", 10)

	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 1 {
		t.Fatal("expected 1 log")
	}

	if res[0].ID != "invX" {
		t.Fatal("unexpected log ID")
	}
}

func TestGetLogsByNameAndVersion_NotFound(t *testing.T) {

	logService := &LogService{
		getFunctionByVersion: func(name, version string) (int, error) {
			return 0, errors.New("not found")
		},
	}

	_, err := logService.GetLogsByNameAndVersion("test", "v2", 10)

	if err == nil {
		t.Fatal("expected error")
	}
}
