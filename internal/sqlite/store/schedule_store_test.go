package store

import (
	"faas-engine-go/internal/sqlite/models"
	"testing"
)

func TestCreateSchedule(t *testing.T) {
	db := setupTestDB(t)

	s := &models.Schedule{
		ID:         "sched-1",
		FunctionID: 1,
		CronExpr:   "@every 1m",
		Payload:    []byte("data"),
	}

	err := CreateSchedule(db, s)
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}

	// verify inserted
	row := db.QueryRow("SELECT COUNT(*) FROM schedules WHERE id=?", s.ID)
	var count int
	_ = row.Scan(&count)

	if count != 1 {
		t.Fatal("schedule not inserted")
	}
}
func TestGetScheduleByID(t *testing.T) {
	db := setupTestDB(t)

	s := &models.Schedule{
		ID:         "sched-1",
		FunctionID: 1,
		CronExpr:   "@every 1m",
		Payload:    []byte("data"),
	}

	_ = CreateSchedule(db, s)

	result, err := GetScheduleByID(db, s.ID)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if result == nil {
		t.Fatal("expected schedule, got nil")
	}

	if result.ID != s.ID {
		t.Fatal("wrong schedule returned")
	}
}
func TestGetScheduleByID_NotFound(t *testing.T) {
	db := setupTestDB(t)

	result, err := GetScheduleByID(db, "non-existent")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if result != nil {
		t.Fatal("expected nil for non-existent schedule")
	}
}
func TestListSchedules(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "test-func", "v1")

	s1 := &models.Schedule{
		ID:         "sched-1",
		FunctionID: 1,
		CronExpr:   "@every 1m",
		Payload:    []byte("data1"),
	}

	s2 := &models.Schedule{
		ID:         "sched-2",
		FunctionID: 1,
		CronExpr:   "@every 2m",
		Payload:    []byte("data2"),
	}

	_ = CreateSchedule(db, s1)
	_ = CreateSchedule(db, s2)

	list, err := ListSchedules(db)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(list))
	}

	if list[0].FunctionName != "test-func" {
		t.Fatal("function name not joined properly")
	}
}
func TestDeleteSchedule(t *testing.T) {
	db := setupTestDB(t)

	s := &models.Schedule{
		ID:         "sched-1",
		FunctionID: 1,
		CronExpr:   "@every 1m",
		Payload:    []byte("data"),
	}

	_ = CreateSchedule(db, s)

	err := DeleteSchedule(db, s.ID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// verify deletion
	row := db.QueryRow("SELECT COUNT(*) FROM schedules WHERE id=?", s.ID)
	var count int
	_ = row.Scan(&count)

	if count != 0 {
		t.Fatal("schedule not deleted")
	}
}
func TestDeleteSchedule_NotFound(t *testing.T) {
	db := setupTestDB(t)

	err := DeleteSchedule(db, "invalid-id")
	if err == nil {
		t.Fatal("expected error for non-existent delete")
	}
}
func TestListSchedulesByFunctionName(t *testing.T) {
	db := setupTestDB(t)

	// 🔥 REQUIRED
	createTestFunction(db, "test-func", "v1")

	s := &models.Schedule{
		ID:         "sched-1",
		FunctionID: 1,
		CronExpr:   "@every 1m",
		Payload:    []byte("data"),
	}

	_ = CreateSchedule(db, s)

	list, err := ListSchedulesByFunctionName(db, "test-func")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(list))
	}
}
