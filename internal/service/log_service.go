package service

import (
	"database/sql"

	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"
)

type LogEntry struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Logs     string `json:"logs"`
	Duration int    `json:"duration"`
}

type LogService struct {
	getLogs func(functionID int, limit int) ([]models.Invocation, error)

	getActiveFunctionID  func(name string) (int, error)
	getFunctionByVersion func(name, version string) (int, error)
}

func NewLogService(db *sql.DB) *LogService {
	return &LogService{

		getLogs: func(functionID int, limit int) ([]models.Invocation, error) {
			return store.GetInvocationLogsByFunction(db, functionID, limit)
		},

		getActiveFunctionID: func(name string) (int, error) {
			fn, err := store.GetActiveFunction(db, name)
			if err != nil {
				return 0, err
			}
			if fn == nil {
				return 0, ErrFunctionNotFound
			}
			return fn.ID, nil
		},

		getFunctionByVersion: func(name, version string) (int, error) {
			fn, err := store.GetFunctionByNameAndVersion(db, name, version)
			if err != nil {
				return 0, err
			}
			if fn == nil {
				return 0, ErrFunctionNotFound
			}
			return fn.ID, nil
		},
	}
}

func (l *LogService) GetLogsByName(functionName string, limit int) ([]LogEntry, error) {

	functionID, err := l.getActiveFunctionID(functionName)
	if err != nil {
		return nil, err
	}

	return l.GetLogs(functionID, limit)
}

func (l *LogService) GetLogsByNameAndVersion(functionName, version string, limit int) ([]LogEntry, error) {

	functionID, err := l.getFunctionByVersion(functionName, version)
	if err != nil {
		return nil, err
	}

	return l.GetLogs(functionID, limit)
}

func (l *LogService) GetLogs(functionID int, limit int) ([]LogEntry, error) {

	invocations, err := l.getLogs(functionID, limit)
	if err != nil {
		return nil, err
	}

	var result []LogEntry

	for _, inv := range invocations {
		result = append(result, LogEntry{
			ID:       inv.ID,
			Status:   inv.Status,
			Logs:     inv.Logs,
			Duration: inv.DurationMs,
		})
	}

	return result, nil
}
