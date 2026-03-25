package models

import "time"

type VersionHistory struct {
	ID          int
	FunctionID  int
	FromVersion string
	ToVersion   string
	TriggeredAt time.Time
}
