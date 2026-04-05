package models

import "time"

type VersionHistory struct {
	ID            string
	FunctionID    string
	FromVersion   string
	ToVersion     string
	TriggeredAt   time.Time
	RequestID     string
	CleanupStatus string
	CleanupError  string
}
