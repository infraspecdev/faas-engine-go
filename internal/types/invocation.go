package types

import "time"

type Invocation struct {
	ID string `json:"id"`

	FunctionName string `json:"function_name"`
	ContainerID  string `json:"container_id"`

	TriggerType string `json:"trigger_type"`

	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`

	DurationMs int64 `json:"duration_ms"`

	RequestPayload  any `json:"request_payload"`
	ResponsePayload any `json:"response_payload"`

	LogsPath string `json:"logs_path"`

	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}
