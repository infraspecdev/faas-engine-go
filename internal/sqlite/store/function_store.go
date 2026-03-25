package store

import (
	"database/sql"
	"faas-engine-go/internal/sqlite/models"
	"fmt"
)

const functionColumns = `
id,
name,
version,
package_checksum,
image,
runtime,
schedule_cron,
endpoint,
status,
created_at
`

func scanFunctionRow(row *sql.Row) (*models.Function, error) {
	var fn models.Function

	err := row.Scan(
		&fn.ID,
		&fn.Name,
		&fn.Version,
		&fn.PackageChecksum,
		&fn.Image,
		&fn.Runtime,
		&fn.ScheduleCron,
		&fn.Endpoint,
		&fn.Status,
		&fn.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fn, nil
}

func scanFunctionFromRows(rows *sql.Rows) (*models.Function, error) {
	var fn models.Function

	err := rows.Scan(
		&fn.ID,
		&fn.Name,
		&fn.Version,
		&fn.PackageChecksum,
		&fn.Image,
		&fn.Runtime,
		&fn.ScheduleCron,
		&fn.Endpoint,
		&fn.Status,
		&fn.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fn, nil
}

func CreateFunction(db *sql.DB, fn *models.Function) error {

	query := `
	INSERT INTO functions (
		name,
		version,
		package_checksum,
		image,
		runtime,
		schedule_cron,
		endpoint,
		status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(
		query,
		fn.Name,
		fn.Version,
		fn.PackageChecksum,
		fn.Image,
		fn.Runtime,
		fn.ScheduleCron,
		fn.Endpoint,
		fn.Status,
	)

	return err
}

func GetFunction(db *sql.DB, name string) (*models.Function, error) {

	query := "SELECT " + functionColumns + " FROM functions WHERE name=?"

	row := db.QueryRow(query, name)

	fn, err := scanFunctionRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return fn, err
}

func GetFunctionByChecksum(db *sql.DB, checksum string) (*models.Function, error) {

	query := "SELECT " + functionColumns + " FROM functions WHERE package_checksum=?"

	row := db.QueryRow(query, checksum)

	fn, err := scanFunctionRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return fn, err
}

func GetActiveFunction(db *sql.DB, name string) (*models.Function, error) {

	query := `
	SELECT ` + functionColumns + `
	FROM functions
	WHERE name=? AND status='active'
	LIMIT 1
	`

	row := db.QueryRow(query, name)

	fn, err := scanFunctionRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return fn, err
}

func GetLatestVersion(db *sql.DB, name string) (string, error) {

	query := `
	SELECT version
	FROM functions
	WHERE name = ?
	ORDER BY id DESC
	LIMIT 1
	`

	var version string

	err := db.QueryRow(query, name).Scan(&version)

	if err == sql.ErrNoRows {
		return "", nil
	}

	return version, err
}

func GetNextVersion(db *sql.DB, name string) (string, error) {

	version, err := GetLatestVersion(db, name)
	if err != nil {
		return "", err
	}

	if version == "" {
		return "v1", nil
	}

	var v int
	_, err = fmt.Sscanf(version, "v%d", &v)
	if err != nil {
		return "", fmt.Errorf("invalid version format: %s", version)
	}

	return fmt.Sprintf("v%d", v+1), nil
}

func DeactivateFunctions(db *sql.DB, name string) error {

	query := `
	UPDATE functions
	SET status = 'inactive'
	WHERE name = ?
	`

	_, err := db.Exec(query, name)
	return err
}

func ListFunctions(db *sql.DB) ([]models.Function, error) {

	query := "SELECT " + functionColumns + " FROM functions"

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []models.Function

	for rows.Next() {

		fn, err := scanFunctionFromRows(rows)
		if err != nil {
			return nil, err
		}

		functions = append(functions, *fn)
	}

	return functions, nil
}

func ListFunctionVersions(db *sql.DB, name string) ([]models.Function, error) {

	query := `
	SELECT ` + functionColumns + `
	FROM functions
	WHERE name = ?
	ORDER BY created_at DESC
	`

	rows, err := db.Query(query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []models.Function

	for rows.Next() {

		fn, err := scanFunctionFromRows(rows)
		if err != nil {
			return nil, err
		}

		functions = append(functions, *fn)
	}

	return functions, nil
}

func DeleteFunction(db *sql.DB, name string) error {

	query := `DELETE FROM functions WHERE name=?`

	_, err := db.Exec(query, name)

	return err
}

func GetFunctionByNameAndVersion(db *sql.DB, name, version string) (*models.Function, error) {

	query := `
	SELECT ` + functionColumns + `
	FROM functions
	WHERE name = ? AND version = ?
	LIMIT 1
	`

	row := db.QueryRow(query, name, version)

	fn, err := scanFunctionRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return fn, err
}

// RollbackToVersion performs an atomic rollback to a specific version.
// It deactivates the current active version and activates the target version,
// then records the rollback event in version_history.
func RollbackToVersion(db *sql.DB, functionName, targetVersion string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get function ID and current active version
	var functionID int
	var currentVersion string
	err = tx.QueryRow(
		"SELECT id, version FROM functions WHERE name=? AND status='active'",
		functionName,
	).Scan(&functionID, &currentVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no active version found for function")
		}
		return "", err
	}

	// Verify target version exists
	var targetID int
	err = tx.QueryRow(
		"SELECT id FROM functions WHERE name=? AND version=?",
		functionName, targetVersion,
	).Scan(&targetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("target version %s not found", targetVersion)
		}
		return "", err
	}

	// Deactivate current version
	_, err = tx.Exec(
		"UPDATE functions SET status='inactive' WHERE name=? AND status='active'",
		functionName,
	)
	if err != nil {
		return "", err
	}

	// Activate target version
	_, err = tx.Exec(
		"UPDATE functions SET status='active' WHERE id=?",
		targetID,
	)
	if err != nil {
		return "", err
	}

	// Record rollback in version_history
	_, err = tx.Exec(
		"INSERT INTO version_history (function_id, from_version, to_version) VALUES (?, ?, ?)",
		functionID, currentVersion, targetVersion,
	)
	if err != nil {
		return "", err
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return currentVersion, nil
}

// GetVersionHistory retrieves rollback history for a function.
func GetVersionHistory(db *sql.DB, functionName string, limit int) ([]models.VersionHistory, error) {
	query := `
	SELECT v.id, v.function_id, v.from_version, v.to_version, v.triggered_at
	FROM version_history v
	JOIN functions f ON v.function_id = f.id
	WHERE f.name = ?
	ORDER BY v.triggered_at DESC
	LIMIT ?
	`

	rows, err := db.Query(query, functionName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.VersionHistory
	for rows.Next() {
		var vh models.VersionHistory
		err := rows.Scan(&vh.ID, &vh.FunctionID, &vh.FromVersion, &vh.ToVersion, &vh.TriggeredAt)
		if err != nil {
			return nil, err
		}
		history = append(history, vh)
	}

	return history, nil
}
