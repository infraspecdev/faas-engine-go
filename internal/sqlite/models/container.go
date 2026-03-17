package models

import "time"

type Container struct {
	ID         string
	FunctionID int
	Status     string
	HostPort   string
	StartedAt  time.Time
	LastUsedAt time.Time
	CreatedAt  time.Time
}
