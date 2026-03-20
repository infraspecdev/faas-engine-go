package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() error {
	var err error

	DB, err = sql.Open("sqlite", "internal/sqlite/faas-engine-go.db")
	if err != nil {
		return err
	}

	err = DB.Ping()
	if err != nil {
		return err
	}

	_, err = DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return err
	}
	DB.Exec("PRAGMA journal_mode = WAL;")
	DB.Exec("PRAGMA busy_timeout = 5000;")
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

		//  SCHEDULES
		`CREATE TABLE IF NOT EXISTS schedules (
			id TEXT PRIMARY KEY,
			function_id INTEGER NOT NULL,
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
		if _, err := DB.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
