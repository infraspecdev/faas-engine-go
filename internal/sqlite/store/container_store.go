package store

import (
	"database/sql"
	"faas-engine-go/internal/sqlite/models"
	"time"
)

const containerColumns = `
id,
function_id,
status,
host_port,
last_used,
created_at
`

func scanContainerRow(row *sql.Row) (*models.Container, error) {
	var c models.Container

	err := row.Scan(
		&c.ID,
		&c.FunctionID,
		&c.Status,
		&c.HostPort,
		&c.LastUsedAt,
		&c.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func scanContainerFromRows(rows *sql.Rows) (*models.Container, error) {
	var c models.Container

	err := rows.Scan(
		&c.ID,
		&c.FunctionID,
		&c.Status,
		&c.HostPort,
		&c.LastUsedAt,
		&c.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ✅ CREATE CONTAINER
func CreateContainer(db *sql.DB, c *models.Container) error {

	query := `
	INSERT INTO containers (
		id,
		function_id,
		status,
		host_port,
		last_used,
		created_at
	) VALUES (?, ?, ?, ?, ?, ?)
	`

	// Retry with exponential backoff for database lock scenarios
	return RetryWithBackoff(func() error {
		_, err := db.Exec(
			query,
			c.ID,
			c.FunctionID,
			c.Status,
			c.HostPort,
			c.LastUsedAt,
			c.CreatedAt,
		)
		return err
	}, 50*time.Millisecond, 5)
}

func GetContainerByID(db *sql.DB, id string) (*models.Container, error) {

	query := "SELECT " + containerColumns + " FROM containers WHERE id=?"

	row := db.QueryRow(query, id)

	c, err := scanContainerRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return c, err
}

func GetContainersByFunction(db *sql.DB, functionID string) ([]models.Container, error) {

	var query string
	var rows *sql.Rows
	var err error

	if functionID == "" {
		// Get ALL containers when functionID is empty (used by container cleanup/spleen)
		query = "SELECT " + containerColumns + " FROM containers"
		rows, err = db.Query(query)
	} else {
		// Get containers for a specific function
		query = "SELECT " + containerColumns + " FROM containers WHERE function_id=?"
		rows, err = db.Query(query, functionID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []models.Container

	for rows.Next() {
		c, err := scanContainerFromRows(rows)
		if err != nil {
			return nil, err
		}
		containers = append(containers, *c)
	}

	return containers, nil
}

func GetFreeContainer(db *sql.DB, functionID string) (*models.Container, error) {
	// Use IMMEDIATE transaction to acquire locks immediately and prevent race conditions
	// when multiple goroutines try to acquire the same free container
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// First, find the free container
	selectQuery := `
	SELECT ` + containerColumns + `
	FROM containers
	WHERE function_id=? AND status='free'
	ORDER BY last_used DESC
	LIMIT 1
	`

	row := tx.QueryRow(selectQuery, functionID)
	c, err := scanContainerRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, tx.Commit()
		}
		return nil, err
	}

	// Update it to busy while still in the transaction
	updateQuery := `
	UPDATE containers
	SET status='busy', last_used=?
	WHERE id=?
	`

	now := time.Now()
	_, err = tx.Exec(updateQuery, now, c.ID)
	if err != nil {
		return nil, err
	}

	// Commit the transaction to release locks
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Update the returned container to reflect the changes
	c.Status = "busy"
	c.LastUsedAt = now

	return c, nil
}

func MarkContainerFree(db *sql.DB, id string) error {

	_, err := db.Exec(`
		UPDATE containers
		SET status='free', last_used=?
		WHERE id=?
	`, time.Now(), id)

	return err
}

func UpdateContainerLastUsed(db *sql.DB, id string) error {

	_, err := db.Exec(`
		UPDATE containers
		SET last_used=?
		WHERE id=?
	`, time.Now(), id)

	return err
}

func CleanupIdleContainers(db *sql.DB, timeout time.Duration, cleanup func(string)) {

	cutoff := time.Now().Add(-timeout)

	rows, err := db.Query(`
		UPDATE containers
		SET status='deleting'
		WHERE id IN (
			SELECT id FROM containers
			WHERE status='free' AND last_used < ?
		)
		RETURNING id
	`, cutoff)

	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		go cleanup(id)
	}
}

func RemoveContainer(db *sql.DB, id string) error {

	_, err := db.Exec(`
		DELETE FROM containers
		WHERE id=?
	`, id)

	return err
}
