package store

import (
	"database/sql"
	"faas-engine-go/internal/sqlite/models"

	"github.com/google/uuid"
)

const scheduleColumns = `
id,
function_id,
cron_expr,
payload,
created_at
`

// ---------- SCAN HELPERS ----------

func scanScheduleRow(row *sql.Row) (*models.Schedule, error) {
	var s models.Schedule

	err := row.Scan(
		&s.ID,
		&s.FunctionID,
		&s.CronExpr,
		&s.Payload,
		&s.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func scanScheduleFromRows(rows *sql.Rows) (*models.Schedule, error) {
	var s models.Schedule

	err := rows.Scan(
		&s.ID,
		&s.FunctionID,
		&s.FunctionName, // ✅ FIX: capture f.name
		&s.CronExpr,
		&s.Payload,
		&s.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

// ---------- CREATE ----------
func CreateSchedule(db *sql.DB, s *models.Schedule) error {

	if s.ID == "" {
		s.ID = uuid.NewString()
	}

	query := `
	INSERT INTO schedules (
		id,
		function_id,
		cron_expr,
		payload
	) VALUES (?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		s.ID,
		s.FunctionID,
		s.CronExpr,
		s.Payload,
	)

	return err
}

// ---------- DELETE ----------
func DeleteSchedule(db *sql.DB, id string) error {

	res, err := db.Exec(`DELETE FROM schedules WHERE id=?`, id)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ---------- GET ONE ----------
func GetScheduleByID(db *sql.DB, id string) (*models.Schedule, error) {

	query := "SELECT " + scheduleColumns + " FROM schedules WHERE id=?"

	row := db.QueryRow(query, id)

	s, err := scanScheduleRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return s, err
}

// ---------- LIST ALL ----------
func ListSchedules(db *sql.DB) ([]models.Schedule, error) {

	query := `
	SELECT 
	s.id,
	s.function_id,
	f.name,
	s.cron_expr,
	s.payload,
	s.created_at
FROM schedules s
JOIN functions f ON s.function_id = f.id
ORDER BY s.created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Schedule

	for rows.Next() {
		s, err := scanScheduleFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ---------- LIST BY FUNCTION ----------
func ListSchedulesByFunctionName(db *sql.DB, functionName string) ([]models.Schedule, error) {

	query := `
	SELECT 
		s.id,
		s.function_id,
		f.name,
		s.cron_expr,
		s.payload,
		s.created_at
	FROM schedules s
	JOIN functions f ON s.function_id = f.id
	WHERE f.name = ?
	ORDER BY s.created_at DESC
	`

	rows, err := db.Query(query, functionName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Schedule

	for rows.Next() {
		s, err := scanScheduleFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
