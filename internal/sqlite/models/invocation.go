package models

import (
	"encoding/json"
	"time"
)

type Invocation struct {
	ID              string
	FunctionID      int
	ContainerID     string
	TriggerType     string
	Status          string
	ExitCode        int
	DurationMs      int
	RequestPayload  json.RawMessage
	ResponsePayload json.RawMessage
	LogsPath        string
	StartedAt       time.Time
	FinishedAt      time.Time
}
