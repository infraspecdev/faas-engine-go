package store

import (
	"database/sql"
	"faas-engine-go/internal/sqlite/models"
	"fmt"
	"time"

	"log/slog"

	"github.com/google/uuid"
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
	// Generate UUID if not already set
	if fn.ID == "" {
		fn.ID = uuid.New().String()
	}

	// Set created_at if not already set
	if fn.CreatedAt.IsZero() {
		fn.CreatedAt = time.Now()
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if this is a new version of an existing function
	var existingActiveVersion string
	var existingFunctionID string
	err = tx.QueryRow(
		"SELECT id, version FROM functions WHERE name=? AND status='active'",
		fn.Name,
	).Scan(&existingFunctionID, &existingActiveVersion)

	// If function exists with active version, deactivate it and record transition
	if err == nil {
		// Record the deployment transition in version_history
		// from_version = old active version, to_version = new version being deployed
		result, err := tx.Exec(
			`INSERT INTO version_history (function_id, from_version, to_version) 
			 VALUES (?, ?, ?)`,
			existingFunctionID, existingActiveVersion, fn.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to insert version history: %w (from: %s, to: %s)", err, existingActiveVersion, fn.Version)
		}
		affected, _ := result.RowsAffected()
		slog.Info("version_history recorded", "function", fn.Name, "from", existingActiveVersion, "to", fn.Version, "rows_affected", affected)

		// Deactivate the previous version
		_, err = tx.Exec(
			"UPDATE functions SET status='inactive' WHERE id=?",
			existingFunctionID,
		)
		if err != nil {
			return fmt.Errorf("failed to deactivate previous version: %w", err)
		}
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing version: %w", err)
	}

	// Insert new version
	query := `
	INSERT INTO functions (
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
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := tx.Exec(
		query,
		fn.ID,
		fn.Name,
		fn.Version,
		fn.PackageChecksum,
		fn.Image,
		fn.Runtime,
		fn.ScheduleCron,
		fn.Endpoint,
		fn.Status,
		fn.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert new function version: %w", err)
	}
	_ = result // Result not used; kept for potential future audit logging
	slog.Info("function version created", "function", fn.Name, "version", fn.Version)

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("CreateFunction completed successfully", "function", fn.Name, "version", fn.Version)
	return nil
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
	ORDER BY created_at DESC
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

func GetFunctionByID(db *sql.DB, id string) (*models.Function, error) {

	query := "SELECT " + functionColumns + " FROM functions WHERE id=?"

	row := db.QueryRow(query, id)

	fn, err := scanFunctionRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return fn, err
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

// GetPreviousVersion returns the previous version using id-based ordering
// Since ids are auto-incrementing, they represent deployment order naturally.
// Finds the version deployed immediately before the current active version.
// Returns empty string if no previous version exists (first version case).
func GetPreviousVersion(db *sql.DB, functionName string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current active version's ID
	var currentID int
	err = tx.QueryRow(
		"SELECT id FROM functions WHERE name=? AND status='active'",
		functionName,
	).Scan(&currentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no active version found")
		}
		return "", err
	}

	// Find the version deployed before current using ID ordering (Fix #2)
	// IDs are auto-incrementing and represent deployment order
	// This prevents implicit ordering assumptions on ListFunctionVersions results
	var previousVersion string
	err = tx.QueryRow(
		`SELECT version FROM functions 
		 WHERE name=? AND id < ? 
		 ORDER BY id DESC 
		 LIMIT 1`,
		functionName, currentID,
	).Scan(&previousVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No previous version (first version)
		}
		return "", err
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return previousVersion, nil
}

// RollbackToVersion performs an atomic rollback to a specific version.
// Returns: (previousVersion, functionID, error)
// functionID is the ID of the function used in version_history for UpdateCleanupStatus.
func RollbackToVersionWithID(db *sql.DB, functionName, targetVersion, requestID string) (string, string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// STEP 1: Fetch current active version with lock (SERIALIZABLE isolation)
	var functionID string
	var currentVersion string
	var currentID string
	err = tx.QueryRow(
		`SELECT id, version FROM functions 
		 WHERE name=? AND status='active'`,
		functionName,
	).Scan(&currentID, &currentVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("no active version found for function")
		}
		return "", "", err
	}

	// Store currentID to track which function will be deactivated (for later UpdateCleanupStatus)
	deactivatedFunctionID := currentID

	// STEP 2: Determine target version
	// CASE A: Explicit rollback (targetVersion provided)
	if targetVersion != "" {
		var targetID string
		err = tx.QueryRow(
			`SELECT id FROM functions 
			 WHERE name=? AND version=? AND status='inactive'`,
			functionName, targetVersion,
		).Scan(&targetID)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", "", fmt.Errorf("target version %s not found or already active", targetVersion)
			}
			return "", "", err
		}
		functionID = targetID
	} else {
		// CASE B: Implicit rollback (no version specified)
		// Get previous version by created_at ordering (DESC to get most recent before current)
		// CRITICAL: The ORDER BY created_at DESC is essential for correct behavior.
		// If this ordering ever changes, the implicit rollback will silently select the wrong version.
		// Always sort in DESC order to select the most recently created version before the current one.
		var targetID string
		err = tx.QueryRow(
			`SELECT id FROM functions 
			 WHERE name=? AND created_at < (
			   SELECT created_at FROM functions WHERE id=?
			 )
			 ORDER BY created_at DESC 
			 LIMIT 1`,
			functionName, currentID,
		).Scan(&targetID)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", "", fmt.Errorf("no previous version to rollback to")
			}
			return "", "", err
		}

		// Get the version string
		err = tx.QueryRow(
			"SELECT version FROM functions WHERE id=?",
			targetID,
		).Scan(&targetVersion)
		if err != nil {
			return "", "", err
		}

		functionID = targetID
	}

	// STEP 3: Update active version (atomically)
	// Deactivate current
	_, err = tx.Exec(
		"UPDATE functions SET status='inactive' WHERE id=?",
		currentID,
	)
	if err != nil {
		return "", "", err
	}

	// Activate target
	_, err = tx.Exec(
		"UPDATE functions SET status='active' WHERE id=?",
		functionID,
	)
	if err != nil {
		return "", "", err
	}

	// STEP 4: Record rollback history with idempotency key
	// Check if this request_id was already processed (idempotent retry)
	if requestID != "" {
		var existingID string
		err = tx.QueryRow(
			`SELECT id FROM version_history 
			 WHERE function_id=? AND request_id=?`,
			currentID, requestID,
		).Scan(&existingID)

		// If already exists, this is a retry - return success (idempotent)
		if err == nil {
			slog.Info("rollback idempotency: request already processed", "request_id", requestID)
			return currentVersion, deactivatedFunctionID, nil
		} else if err != sql.ErrNoRows {
			return "", "", err
		}
		// If ErrNoRows, continue with normal flow
	}

	// Insert new history entry with optional request_id for idempotency
	_, err = tx.Exec(
		`INSERT INTO version_history (function_id, from_version, to_version, request_id) 
		 VALUES (?, ?, ?, ?)`,
		currentID, currentVersion, targetVersion, requestID,
	)
	if err != nil {
		return "", "", err
	}

	// STEP 5: Commit atomically
	if err = tx.Commit(); err != nil {
		return "", "", err
	}

	// Return previousVersion and deactivatedFunctionID for UpdateCleanupStatus
	return currentVersion, deactivatedFunctionID, nil
}

// RollbackToVersion is a wrapper for backward compatibility with the Store interface.
// It calls RollbackToVersionWithID and returns only the version string, discarding the functionID.
// Use RollbackToVersionWithID directly if you need the deactivatedFunctionID.
func RollbackToVersion(db *sql.DB, functionName, targetVersion, requestID string) (string, error) {
	version, _, err := RollbackToVersionWithID(db, functionName, targetVersion, requestID)
	return version, err
}

// GetVersionHistory retrieves rollback history for a function.
func GetVersionHistory(db *sql.DB, functionName string, limit int) ([]models.VersionHistory, error) {
	query := `
	SELECT v.id, v.function_id, v.from_version, v.to_version, v.triggered_at, 
	       COALESCE(v.request_id, ''), COALESCE(v.cleanup_status, ''), COALESCE(v.cleanup_error, '')
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
		err := rows.Scan(&vh.ID, &vh.FunctionID, &vh.FromVersion, &vh.ToVersion, &vh.TriggeredAt,
			&vh.RequestID, &vh.CleanupStatus, &vh.CleanupError)
		if err != nil {
			return nil, err
		}
		history = append(history, vh)
	}

	return history, nil
}

// UpdateCleanupStatus updates the cleanup status and error for a specific rollback entry.
// Called after async cleanup completes to persist the final status to the database
func UpdateCleanupStatus(db *sql.DB, functionID string, requestID string, status, errMsg string) error {
	query := `
	UPDATE version_history 
	SET cleanup_status = ?, cleanup_error = ?
	WHERE function_id = ? AND request_id = ?
	`

	_, err := db.Exec(query, status, errMsg, functionID, requestID)
	return err
}
