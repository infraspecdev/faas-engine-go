package models

import "time"

type Function struct {
	ID              int
	Name            string
	Version         string
	PackageChecksum string
	Image           string
	Runtime         string
	ScheduleCron    string
	Endpoint        string // great.localhost
	Status          string
	CreatedAt       time.Time
}
