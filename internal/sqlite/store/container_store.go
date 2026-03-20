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

func CreateContainer(db *sql.DB, c *models.Container) error {

	query := `
	INSERT INTO containers (
		id,
		function_id,
		status,
		host_port,
		last_used
	) VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		c.ID,
		c.FunctionID,
		c.Status,
		c.HostPort,
		c.LastUsedAt,
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

func GetFreeContainer(db *sql.DB, functionID int) (*models.Container, error) {

	query := `
	SELECT ` + containerColumns + `
	FROM containers
	WHERE function_id=? AND status='free'
	ORDER BY last_used ASC
	LIMIT 1
	`

	row := db.QueryRow(query, functionID)

	c, err := scanContainerRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return c, err
}

func MarkContainerBusy(db *sql.DB, id string) error {

	query := `
	UPDATE containers
	SET status='busy'
	WHERE id=?
	`

	_, err := db.Exec(query, id)
	return err
}

func MarkContainerFree(db *sql.DB, id string) error {

	query := `
	UPDATE containers
	SET status='free', last_used=CURRENT_TIMESTAMP
	WHERE id=?
	`

	_, err := db.Exec(query, id)
	return err
}

func UpdateContainerLastUsed(db *sql.DB, id string) error {

	query := `
	UPDATE containers
	SET last_used=CURRENT_TIMESTAMP
	WHERE id=?
	`

	_, err := db.Exec(query, id)
	return err
}

func CleanupIdleContainers(timeout time.Duration, cleanup func(string)) {

	rows, err := sqlite.DB.Query(`
		UPDATE containers
		SET status='deleting'
		WHERE id IN (
			SELECT id FROM containers
			WHERE status='free' AND last_used < ?
		)
		RETURNING id
	`, time.Now().Add(-timeout))

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

	query := `
	DELETE FROM containers
	WHERE id=?
	`

	_, err := db.Exec(query, id)
	return err
}
