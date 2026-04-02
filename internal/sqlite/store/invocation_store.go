package store

import (
	"database/sql"
	"encoding/json"
	"faas-engine-go/internal/sqlite/models"
	"time"

	"github.com/google/uuid"
)

const invocationColumns = `
id,
function_id,
container_id,
trigger_type,
status,
exit_code,
duration_ms,
request_payload,
response_payload,
logs,
started_at,
finished_at
`

func scanInvocationRow(row *sql.Row) (*models.Invocation, error) {
	var inv models.Invocation

	err := row.Scan(
		&inv.ID,
		&inv.FunctionID,
		&inv.ContainerID,
		&inv.TriggerType,
		&inv.Status,
		&inv.ExitCode,
		&inv.DurationMs,
		&inv.RequestPayload,
		&inv.ResponsePayload,
		&inv.Logs,
		&inv.StartedAt,
		&inv.FinishedAt,
	)

	if err != nil {
		return nil, err
	}

	return &inv, nil
}

func scanInvocationFromRows(rows *sql.Rows) (*models.Invocation, error) {
	var inv models.Invocation

	var logs sql.NullString
	var response sql.NullString
	var finishedAt sql.NullTime

	err := rows.Scan(
		&inv.ID,
		&inv.FunctionID,
		&inv.ContainerID,
		&inv.TriggerType,
		&inv.Status,
		&inv.ExitCode,
		&inv.DurationMs,
		&inv.RequestPayload,
		&response,
		&logs,
		&inv.StartedAt,
		&finishedAt,
	)
	if err != nil {
		return nil, err
	}

	if logs.Valid {
		inv.Logs = logs.String
	} else {
		inv.Logs = ""
	}

	if response.Valid {
		inv.ResponsePayload = []byte(response.String)
	}

	if finishedAt.Valid {
		inv.FinishedAt = finishedAt.Time
	}

	return &inv, nil
}

func CreateInvocation(db *sql.DB, inv *models.Invocation) error {

	if inv.ID == "" {
		inv.ID = uuid.NewString()
	}

	query := `
	INSERT INTO invocations (
		id,
		function_id,
		container_id,
		trigger_type,
		status,
		exit_code,
		duration_ms,
		request_payload,
		response_payload,
		logs,
		started_at,
		finished_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Retry with exponential backoff for database lock scenarios
	return RetryWithBackoff(func() error {
		_, err := db.Exec(
			query,
			inv.ID,
			inv.FunctionID,
			inv.ContainerID,
			inv.TriggerType,
			inv.Status,
			inv.ExitCode,
			inv.DurationMs,
			inv.RequestPayload,
			inv.ResponsePayload,
			inv.Logs,
			inv.StartedAt,
			inv.FinishedAt,
		)
		return err
	}, 50*time.Millisecond, 5)
}

func MarkInvocationRunning(db *sql.DB, id string, containerID string) error {

	query := `
	UPDATE invocations
	SET status='running',
	    container_id=?,
	    started_at=?
	WHERE id=?
	`

	_, err := db.Exec(query, containerID, time.Now(), id)
	return err
}

func CompleteInvocation(
	db *sql.DB,
	id string,
	status string,
	exitCode int,
	response json.RawMessage,
	logs string,
	startedAt time.Time,
) error {

	duration := int(time.Since(startedAt).Milliseconds())
	query := `
	UPDATE invocations
	SET status=?,
	    exit_code=?,
	    duration_ms=?,
	    response_payload=?,
	    logs=?,
	    finished_at=?
	WHERE id=?
	`

	_, err := db.Exec(
		query,
		status,
		exitCode,
		duration,
		response,
		logs,
		time.Now(),
		id,
	)

	return err
}

// CompleteInvocationAndMarkFree atomically completes an invocation and marks container as free
// This consolidates two write operations into a single transaction for better concurrency
func CompleteInvocationAndMarkFree(
	db *sql.DB,
	id string,
	status string,
	exitCode int,
	response json.RawMessage,
	logs string,
	startedAt time.Time,
	containerID string,
	shouldMarkFree bool,
) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Update invocation
	duration := int(time.Since(startedAt).Milliseconds())
	invQuery := `
	UPDATE invocations
	SET status=?,
	    exit_code=?,
	    duration_ms=?,
	    response_payload=?,
	    logs=?,
	    finished_at=?
	WHERE id=?
	`

	if _, err := tx.Exec(
		invQuery,
		status,
		exitCode,
		duration,
		response,
		logs,
		time.Now(),
		id,
	); err != nil {
		return err
	}

	// Mark container as free if requested
	if shouldMarkFree && containerID != "" {
		containerQuery := `
		UPDATE containers
		SET status='free',
		    last_used=?
		WHERE id=?
		`
		if _, err := tx.Exec(containerQuery, time.Now(), containerID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func GetInvocationByID(db *sql.DB, id string) (*models.Invocation, error) {

	query := "SELECT " + invocationColumns + " FROM invocations WHERE id=?"

	row := db.QueryRow(query, id)

	inv, err := scanInvocationRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return inv, err
}

func ListInvocationsByFunction(db *sql.DB, functionID string, limit int) ([]models.Invocation, error) {

	query := `
	SELECT ` + invocationColumns + `
	FROM invocations
	WHERE function_id = ?
	ORDER BY started_at DESC
	LIMIT ?
	`

	rows, err := db.Query(query, functionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Invocation

	for rows.Next() {

		inv, err := scanInvocationFromRows(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, *inv)
	}

	return result, nil
}

func ListInvocationsByStatus(db *sql.DB, status string, limit int) ([]models.Invocation, error) {

	query := `
	SELECT ` + invocationColumns + `
	FROM invocations
	WHERE status = ?
	ORDER BY started_at DESC
	LIMIT ?
	`

	rows, err := db.Query(query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Invocation

	for rows.Next() {

		inv, err := scanInvocationFromRows(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, *inv)
	}

	return result, nil
}

func GetInvocationLogsByFunction(db *sql.DB, functionID string, limit int) ([]models.Invocation, error) {

	query := `
	SELECT ` + invocationColumns + `
	FROM (
		SELECT ` + invocationColumns + `
		FROM invocations
		WHERE function_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	)
	ORDER BY started_at ASC
	`

	rows, err := db.Query(query, functionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Invocation

	for rows.Next() {
		inv, err := scanInvocationFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *inv)
	}

	return result, nil
}
