package store

import (
	"database/sql"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	sqlite.DB = db

	if err := sqlite.InitTables(); err != nil {
		t.Fatal(err)
	}

	return db
}

func TestCreateAndGetFunction(t *testing.T) {
	db := setupTestDB(t)

	fn := &models.Function{
		Name:      "calc",
		Version:   "v1",
		Status:    "active",
		CreatedAt: time.Now(),
	}

	err := CreateFunction(db, fn)
	if err != nil {
		t.Fatal(err)
	}

	res, err := GetFunction(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || res.Name != "calc" {
		t.Fatalf("expected calc, got %+v", res)
	}
}

func TestGetFunction_NotFound(t *testing.T) {
	db := setupTestDB(t)

	res, err := GetFunction(db, "unknown")
	if err != nil {
		t.Fatal(err)
	}

	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}
}

func TestGetActiveFunction(t *testing.T) {
	db := setupTestDB(t)

	fn := &models.Function{
		Name:    "calc",
		Version: "v1",
		Status:  "active",
	}

	_ = CreateFunction(db, fn)

	res, err := GetActiveFunction(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || res.Status != "active" {
		t.Fatalf("expected active function, got %+v", res)
	}
}

func TestGetLatestVersion(t *testing.T) {
	db := setupTestDB(t)

	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "v1"})
	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "v2"})

	version, err := GetLatestVersion(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if version != "v2" {
		t.Fatalf("expected v2, got %s", version)
	}
}

func TestGetNextVersion(t *testing.T) {
	db := setupTestDB(t)

	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "v2"})

	version, err := GetNextVersion(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if version != "v3" {
		t.Fatalf("expected v3, got %s", version)
	}
}

func TestGetNextVersion_First(t *testing.T) {
	db := setupTestDB(t)

	version, err := GetNextVersion(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if version != "v1" {
		t.Fatalf("expected v1, got %s", version)
	}
}

func TestGetNextVersion_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)

	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "bad"})

	_, err := GetNextVersion(db, "calc")
	if err == nil {
		t.Fatalf("expected error for invalid version format")
	}
}

func TestListFunctions(t *testing.T) {
	db := setupTestDB(t)

	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "v1"})
	_ = CreateFunction(db, &models.Function{Name: "auth", Version: "v1"})

	functions, err := ListFunctions(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(functions))
	}
}

func TestListFunctionVersions(t *testing.T) {
	db := setupTestDB(t)

	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "v1"})
	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "v2"})

	functions, err := ListFunctionVersions(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if len(functions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(functions))
	}
}

func TestDeactivateFunctions(t *testing.T) {
	db := setupTestDB(t)

	_ = CreateFunction(db, &models.Function{
		Name:    "calc",
		Version: "v1",
		Status:  "active",
	})

	err := DeactivateFunctions(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	fn, _ := GetFunction(db, "calc")

	if fn.Status != "inactive" {
		t.Fatalf("expected inactive, got %s", fn.Status)
	}
}

func TestDeleteFunction(t *testing.T) {
	db := setupTestDB(t)

	_ = CreateFunction(db, &models.Function{Name: "calc", Version: "v1"})

	err := DeleteFunction(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	res, _ := GetFunction(db, "calc")

	if res != nil {
		t.Fatalf("expected nil after delete, got %+v", res)
	}
}
