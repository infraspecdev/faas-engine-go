package service

import (
	"sync"
	"testing"

	"faas-engine-go/internal/db"
	"faas-engine-go/internal/types"
)

// resetDB clears the in-memory db state between tests.
// Add this function to your db package:
//
//	func Reset() {
//	    scheduleMu.Lock()
//	    defer scheduleMu.Unlock()
//	    scheduleStore = make(map[string]types.Schedule)
//	}
func resetDB() {
	db.Reset()
}

func newTestScheduler() *SchedulerService {
	// nil invoker is safe for tests that don't trigger cron execution
	svc := NewSchedulerService(nil)
	svc.Start()
	return svc
}

// --- AddSchedule ---

func TestAddSchedule_Success(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	err := svc.AddSchedule("my-func", "*/5 * * * *", []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	schedules := svc.ListSchedules()
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}
	if schedules[0].FunctionName != "my-func" {
		t.Errorf("expected function name 'my-func', got '%s'", schedules[0].FunctionName)
	}
	if schedules[0].CronExpr != "*/5 * * * *" {
		t.Errorf("expected cron '*/5 * * * *', got '%s'", schedules[0].CronExpr)
	}
}

func TestAddSchedule_EmptyFunctionName(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	err := svc.AddSchedule("", "*/5 * * * *", nil)
	if err == nil {
		t.Fatal("expected error for empty function name, got nil")
	}
}

func TestAddSchedule_EmptyCronExpr(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	err := svc.AddSchedule("my-func", "", nil)
	if err == nil {
		t.Fatal("expected error for empty cron expression, got nil")
	}
}

func TestAddSchedule_InvalidCronExpr(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	err := svc.AddSchedule("my-func", "not-a-cron", nil)
	if err == nil {
		t.Fatal("expected error for invalid cron expression, got nil")
	}
}

func TestAddSchedule_AssignsUniqueIDs(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	_ = svc.AddSchedule("fn-a", "*/1 * * * *", nil)
	_ = svc.AddSchedule("fn-b", "*/2 * * * *", nil)

	schedules := svc.ListSchedules()
	if len(schedules) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(schedules))
	}
	if schedules[0].ID == schedules[1].ID {
		t.Error("expected unique IDs for each schedule")
	}
}

// --- DeleteSchedule ---

func TestDeleteSchedule_Success(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	_ = svc.AddSchedule("my-func", "*/5 * * * *", nil)
	schedules := svc.ListSchedules()
	id := schedules[0].ID

	err := svc.DeleteSchedule(id)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	remaining := svc.ListSchedules()
	if len(remaining) != 0 {
		t.Errorf("expected 0 schedules after delete, got %d", len(remaining))
	}
}

func TestDeleteSchedule_NotFound(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	err := svc.DeleteSchedule("non-existent-id")
	if err == nil {
		t.Fatal("expected error for missing schedule, got nil")
	}
}

func TestDeleteSchedule_RemovesFromCronEntries(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	_ = svc.AddSchedule("my-func", "*/5 * * * *", nil)
	id := svc.ListSchedules()[0].ID

	_ = svc.DeleteSchedule(id)

	// Deleting again should fail — confirming it was removed from cronEntries
	err := svc.DeleteSchedule(id)
	if err == nil {
		t.Fatal("expected error on second delete, got nil")
	}
}

// --- ListSchedules ---

func TestListSchedules_Empty(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	schedules := svc.ListSchedules()
	if len(schedules) != 0 {
		t.Errorf("expected empty list, got %d", len(schedules))
	}
}

func TestListSchedules_ReturnsAll(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	_ = svc.AddSchedule("fn-a", "*/1 * * * *", nil)
	_ = svc.AddSchedule("fn-b", "*/2 * * * *", nil)
	_ = svc.AddSchedule("fn-c", "*/3 * * * *", nil)

	schedules := svc.ListSchedules()
	if len(schedules) != 3 {
		t.Errorf("expected 3 schedules, got %d", len(schedules))
	}
}

// --- GetSchedulesByFunction ---

func TestGetSchedulesByFunction_Match(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	_ = svc.AddSchedule("target-fn", "*/1 * * * *", nil)
	_ = svc.AddSchedule("target-fn", "*/2 * * * *", nil)
	_ = svc.AddSchedule("other-fn", "*/3 * * * *", nil)

	results := svc.GetSchedulesByFunction("target-fn")
	if len(results) != 2 {
		t.Errorf("expected 2 schedules for 'target-fn', got %d", len(results))
	}
	for _, s := range results {
		if s.FunctionName != "target-fn" {
			t.Errorf("unexpected function name '%s' in results", s.FunctionName)
		}
	}
}

func TestGetSchedulesByFunction_NoMatch(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	_ = svc.AddSchedule("other-fn", "*/1 * * * *", nil)

	results := svc.GetSchedulesByFunction("ghost-fn")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// --- Start (loads persisted schedules) ---

func TestStart_LoadsPersistedSchedules(t *testing.T) {
	resetDB()

	// Pre-populate db directly (simulates server restart with existing schedules)
	db.AddSchedule(types.Schedule{
		ID:           "pre-existing-1",
		FunctionName: "persisted-fn",
		CronExpr:     "*/5 * * * *",
	})

	svc := NewSchedulerService(nil)
	svc.Start()

	if _, ok := svc.cronEntries["pre-existing-1"]; !ok {
		t.Error("expected pre-existing schedule to be registered in cronEntries after Start()")
	}
}

// --- Concurrency ---

func TestAddDeleteSchedule_Concurrent(t *testing.T) {
	resetDB()
	svc := newTestScheduler()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.AddSchedule("concurrent-fn", "*/1 * * * *", nil)
		}()
	}
	wg.Wait()

	schedules := svc.ListSchedules()
	if len(schedules) != 10 {
		t.Errorf("expected 10 schedules after concurrent adds, got %d", len(schedules))
	}
}
