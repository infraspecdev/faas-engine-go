package models

import "time"

type Container struct {
	ID         string
	FunctionID string
	Status     string
	HostPort   string
	StartedAt  time.Time
	LastUsedAt time.Time
	CreatedAt  time.Time
}
