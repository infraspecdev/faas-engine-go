package store

import (
	"database/sql"
	"faas-engine-go/internal/sqlite"
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

	_, err := db.Exec(
		query,
		c.ID,
		c.FunctionID,
		c.Status,
		c.HostPort,
		c.LastUsedAt,
		time.Now(),
	)

	return err
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

func GetContainersByFunction(db *sql.DB, functionID int) ([]models.Container, error) {

	query := "SELECT " + containerColumns + " FROM containers WHERE function_id=?"

	rows, err := db.Query(query, functionID)
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

func AcquireFreeContainer(db *sql.DB, functionID int) (*models.Container, error) {

	query := `
	UPDATE containers
	SET status='busy', last_used=?
	WHERE id = (
		SELECT id FROM containers
		WHERE function_id=? AND status='free'
		ORDER BY last_used DESC
		LIMIT 1
	)
	RETURNING ` + containerColumns

	row := db.QueryRow(query, time.Now(), functionID)

	c, err := scanContainerRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return c, err
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

func CleanupIdleContainers(timeout time.Duration, cleanup func(string)) {

	cutoff := time.Now().Add(-timeout)

	rows, err := sqlite.DB.Query(`
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
