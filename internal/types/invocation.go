package types

import "time"

type Invocation struct {
	ID string

	FunctionName string
	ContainerID  string

	TriggerType string // "http" | "cron" | "manual"

	Status   string // "running" | "success" | "failed"
	ExitCode int

	DurationMs int64

	RequestPayload  []byte // JSON
	ResponsePayload []byte // JSON

	LogsPath string

	StartedAt  time.Time
	FinishedAt time.Time
}
