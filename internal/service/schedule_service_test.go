package service

import (
	"context"
	"testing"

	"faas-engine-go/internal/sqlite/models"
)

type MockInvoker struct {
	Called bool
	Name   string
	Data   string
	Err    error

	InvokeFunc func(ctx context.Context, name string, payload []byte, source string) (any, error)
}

func (m *MockInvoker) Invoke(ctx context.Context, functionName string, payload []byte, triggerType string) (any, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, functionName, payload, triggerType)
	}
	m.Called = true
	m.Name = functionName
	m.Data = string(payload)
	return "ok", m.Err
}

// ---------- TEST: NewSchedulerService ----------
func TestNewSchedulerService(t *testing.T) {
	mock := &MockInvoker{}
	s := NewSchedulerService(mock, nil)

	if s.cron == nil {
		t.Fatal("cron should not be nil")
	}
	if s.invoker == nil {
		t.Fatal("invoker should not be nil")
	}
	if s.entries == nil {
		t.Fatal("entries map should not be nil")
	}
	if s.scheduleSemaphores == nil {
		t.Fatal("scheduleSemaphores map should not be nil")
	}
}

// ---------- TEST: RegisterSchedule ----------
func TestRegisterSchedule(t *testing.T) {
	mock := &MockInvoker{}
	s := NewSchedulerService(mock, nil)

	sch := models.Schedule{
		ID:         "1",
		CronExpr:   "* * * * * *",  // Every second (6-field format)
		FunctionID: "fn-uuid-123",  // ✅ string UUID
		Payload:    []byte("test"), // ✅ []byte
	}
	err := s.RegisterSchedule(sch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := s.entries["1"]; !exists {
		t.Fatal("schedule not registered")
	}
}

// ---------- TEST: Duplicate Schedule ----------
func TestRegisterDuplicateSchedule(t *testing.T) {
	mock := &MockInvoker{}
	s := NewSchedulerService(mock, nil)

	sch := models.Schedule{
		ID:         "1",
		CronExpr:   "* * * * * *",  // Every second (6-field format)
		FunctionID: "fn-uuid-123",  // ✅ string UUID
		Payload:    []byte("test"), // ✅ []byte
	}

	_ = s.RegisterSchedule(sch)
	_ = s.RegisterSchedule(sch)

	if len(s.entries) != 1 {
		t.Fatal("duplicate schedule should not be added")
	}
}

// ---------- TEST: Invalid Cron ----------
func TestRegisterInvalidCron(t *testing.T) {
	mock := &MockInvoker{}
	s := NewSchedulerService(mock, nil)

	sch := models.Schedule{
		ID:       "1",
		CronExpr: "invalid-cron",
	}

	err := s.RegisterSchedule(sch)
	if err == nil {
		t.Fatal("expected error for invalid cron")
	}
}

// ---------- TEST: RemoveSchedule ----------
func TestRemoveSchedule(t *testing.T) {
	mock := &MockInvoker{}
	s := NewSchedulerService(mock, nil)

	sch := models.Schedule{
		ID:       "1",
		CronExpr: "@every 1s",
	}

	_ = s.RegisterSchedule(sch)
	s.RemoveSchedule("1")

	if _, exists := s.entries["1"]; exists {
		t.Fatal("schedule not removed")
	}
}
