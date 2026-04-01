package store

import (
	"database/sql"
	"testing"
	"time"

	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	sqlite.SetDB(db)

	t.Cleanup(func() {
		db.Close()
	})

	if err := sqlite.InitTables(); err != nil {
		t.Fatalf("failed to init tables: %v", err)
	}

	return db
}

// 🔹 helper to avoid NULL issues everywhere
func createTestFunction(db *sql.DB, name, version string) string {
	fn := &models.Function{
		Name:            name,
		Version:         version,
		PackageChecksum: "chk",
		Image:           "img",
		Runtime:         "node",
		ScheduleCron:    "",
		Endpoint:        "/test",
		Status:          "active",
		CreatedAt:       time.Now(),
	}
	_ = CreateFunction(db, fn)
	// Small delay to ensure different timestamps for subsequent calls
	time.Sleep(1 * time.Millisecond)
	return fn.ID
}
