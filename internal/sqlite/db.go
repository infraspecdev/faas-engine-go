package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"faas-engine-go/internal/config"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func InitDB() (*sql.DB, error) {
	dbURL := os.Getenv("DB_URL")

	if dbURL == "" {
		dbURL = "internal/sqlite/faas-engine-go.db"
	}

	if strings.HasPrefix(dbURL, "/") {
		dir := filepath.Dir(dbURL)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db dir: %w", err)
		}
	}

	var err error
	db, err = sql.Open("sqlite", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d;", config.SQLiteBusyTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Optimize for concurrent writes
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(0) // No lifetime limit (SQLite doesn't need it)
	return db, nil
}

func SetDB(conn *sql.DB) {
	db = conn
}

func GetDB() *sql.DB {
	return db
}

func InitTables() error {

	queries := []string{

		//  FUNCTIONS
		`CREATE TABLE IF NOT EXISTS functions (
			id TEXT PRIMARY KEY,
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
			function_id TEXT,
			status TEXT,
			host_port TEXT,
			last_used TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(function_id) REFERENCES functions(id)
		);`,

		//  INVOCATIONS
		`CREATE TABLE IF NOT EXISTS invocations (
			id TEXT PRIMARY KEY,
			function_id TEXT,
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

		//  SCHEDULES
		`CREATE TABLE IF NOT EXISTS schedules (
			id TEXT PRIMARY KEY,
			function_id TEXT NOT NULL,
			cron_expr TEXT NOT NULL,
			payload TEXT, -- changed from BLOB → TEXT
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

			FOREIGN KEY(function_id) REFERENCES functions(id) ON DELETE CASCADE
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

		// SCHEDULE INDEXES
		`CREATE INDEX IF NOT EXISTS idx_schedules_function_id 
		ON schedules(function_id);`,

		`CREATE INDEX IF NOT EXISTS idx_schedules_cron 
		ON schedules(cron_expr);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
