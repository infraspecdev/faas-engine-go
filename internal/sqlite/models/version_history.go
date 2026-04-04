package models

import "time"

type VersionHistory struct {
	ID            int
	FunctionID    int
	FromVersion   string
	ToVersion     string
	TriggeredAt   time.Time
	RequestID     string // Fix #6: For idempotency - prevents duplicate entries on retry
	CleanupStatus string // Fix #7: Track cleanup status
	CleanupError  string // Fix #7: Track cleanup errors
}
