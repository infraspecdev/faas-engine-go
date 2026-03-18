package types

type Schedule struct {
	ID           string `json:"id"`
	FunctionName string `json:"function"`
	CronExpr     string `json:"cron"`
	Payload      []byte `json:"payload"`
}
