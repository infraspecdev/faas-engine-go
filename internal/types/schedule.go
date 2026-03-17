package types

type Schedule struct {
	ID           string
	FunctionName string
	CronExpr     string
	Payload      []byte
}
