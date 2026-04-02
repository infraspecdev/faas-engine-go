package models

import "time"

type Function struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	PackageChecksum string    `json:"package_checksum"`
	Image           string    `json:"image"`
	Runtime         string    `json:"runtime"`
	ScheduleCron    string    `json:"schedule_cron"`
	Endpoint        string    `json:"endpoint"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}
