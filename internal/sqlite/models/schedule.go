package models

import "time"

type Schedule struct {
	ID           string    `json:"id"`
	FunctionID   string    `json:"function_id"`
	FunctionName string    `json:"function_name"`
	CronExpr     string    `json:"cron"`
	Payload      []byte    `json:"payload"`
	CreatedAt    time.Time `json:"created_at"`
}
