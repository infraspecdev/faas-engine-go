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

// -----------------------------
// 🔹 UPDATE
// -----------------------------

func DeactivateFunctions(db *sql.DB, name string) error {

	query := `
	UPDATE functions
	SET status = 'inactive'
	WHERE name = ?
	`

	_, err := db.Exec(query, name)
	return err
}

// -----------------------------
// 🔹 LIST
// -----------------------------

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

func GetFunctionByID(db *sql.DB, id int) (*models.Function, error) {

	query := "SELECT " + functionColumns + " FROM functions WHERE id=?"

	row := db.QueryRow(query, id)

	fn, err := scanFunctionRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return fn, err
}
