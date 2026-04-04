package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() error {
	dbURL := os.Getenv("DB_URL")

	if dbURL == "" {
		dbURL = "internal/sqlite/faas-engine-go.db"
	}

	if strings.HasPrefix(dbURL, "/") {
		dir := filepath.Dir(dbURL)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create db dir: %w", err)
		}
	}

	var err error
	DB, err = sql.Open("sqlite", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	if _, err := DB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("failed to enable WAL: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping db: %w", err)
	}

	fmt.Println("Using DB:", dbURL)

	return nil
}

func InitTables() error {

	queries := []string{

		//  FUNCTIONS
		`CREATE TABLE IF NOT EXISTS functions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			version TEXT,
			package_checksum TEXT,
			image TEXT,
			runtime TEXT,
			schedule_cron TEXT,
			endpoint TEXT,
			status TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		//  CONTAINERS
		`CREATE TABLE IF NOT EXISTS containers (
			id TEXT PRIMARY KEY,
			function_id INTEGER,
			status TEXT,
			host_port TEXT,
			last_used TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(function_id) REFERENCES functions(id)
		);`,

		//  INVOCATIONS
		`CREATE TABLE IF NOT EXISTS invocations (
			id TEXT PRIMARY KEY,
			function_id INTEGER,
			container_id TEXT,
			trigger_type TEXT,
			status TEXT,
			exit_code INTEGER,
			duration_ms INTEGER,
			request_payload TEXT,	
			response_payload TEXT,
			logs TEXT,
			started_at DATETIME,
			finished_at DATETIME
		);`,

		//  INDEXES
		`CREATE INDEX IF NOT EXISTS idx_functions_name 
		ON functions(name);`,

		`CREATE INDEX IF NOT EXISTS idx_functions_name_status 
		ON functions(name, status);`,

		`CREATE INDEX IF NOT EXISTS idx_functions_name_created 
		ON functions(name, created_at DESC);`,

		`CREATE INDEX IF NOT EXISTS idx_containers_function_id 
		ON containers(function_id);`,

		`CREATE INDEX IF NOT EXISTS idx_containers_status 
		ON containers(status);`,

		`CREATE INDEX IF NOT EXISTS idx_containers_fn_status 
		ON containers(function_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_containers_last_used 
		ON containers(last_used);`,

		`CREATE INDEX IF NOT EXISTS idx_invocations_function_id 
		ON invocations(function_id);`,

		`CREATE INDEX IF NOT EXISTS idx_invocations_status 
		ON invocations(status);`,

		//  VERSION_HISTORY
		`CREATE TABLE IF NOT EXISTS version_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			function_id INTEGER,
			from_version TEXT,
			to_version TEXT,
			triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			cleanup_status TEXT DEFAULT 'pending',
			cleanup_error TEXT,
			FOREIGN KEY(function_id) REFERENCES functions(id)
		);	`,

		// VERSION_HISTORY: Add cleanup columns if they don't exist
		`ALTER TABLE version_history ADD COLUMN cleanup_status TEXT DEFAULT 'pending';`,
		`ALTER TABLE version_history ADD COLUMN cleanup_error TEXT;`,

		// Fix #6: Add idempotency key for retry deduplication
		// Prevents duplicate history entries when rollback is retried
		`ALTER TABLE version_history ADD COLUMN request_id TEXT;`,

		`CREATE INDEX IF NOT EXISTS idx_version_history_function_id
		ON version_history(function_id);`,

		`CREATE INDEX IF NOT EXISTS idx_version_history_triggered_at
		ON version_history(triggered_at DESC);`,

		// Fix #6: Unique index for idempotency - prevents duplicate entries on retry
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_version_history_idempotency
		ON version_history(function_id, request_id) WHERE request_id IS NOT NULL;`,

		// ROLLBACK_STACK: LIFO stack for version rollbacks
		`CREATE TABLE IF NOT EXISTS rollback_stack (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			function_name TEXT NOT NULL,
			version TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE INDEX IF NOT EXISTS idx_rollback_stack_function_name
		ON rollback_stack(function_name);`,

		`CREATE INDEX IF NOT EXISTS idx_rollback_stack_created_at
		ON rollback_stack(function_name, created_at DESC);`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return err
		}
	}

	return nil
}
