package store

import (
	"testing"
)

// ---------- CREATE + GET ----------
func TestCreateAndGetFunction(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v1")

	res, err := GetFunction(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || res.Name != "calc" {
		t.Fatalf("expected calc, got %+v", res)
	}
}

// ---------- NOT FOUND ----------
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

// ---------- ACTIVE FUNCTION ----------
func TestGetActiveFunction(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v1")

	res, err := GetActiveFunction(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || res.Status != "active" {
		t.Fatalf("expected active function, got %+v", res)
	}
}

// ---------- LATEST VERSION ----------
func TestGetLatestVersion(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v1")
	createTestFunction(db, "calc", "v2")

	version, err := GetLatestVersion(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if version != "v2" {
		t.Fatalf("expected v2, got %s", version)
	}
}

// ---------- NEXT VERSION ----------
func TestGetNextVersion(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v2")

	version, err := GetNextVersion(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if version != "v3" {
		t.Fatalf("expected v3, got %s", version)
	}
}

// ---------- FIRST VERSION ----------
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

// ---------- INVALID VERSION ----------
func TestGetNextVersion_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "bad")

	_, err := GetNextVersion(db, "calc")
	if err == nil {
		t.Fatalf("expected error for invalid version format")
	}
}

// ---------- LIST FUNCTIONS ----------
func TestListFunctions(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v1")
	createTestFunction(db, "auth", "v1")

	functions, err := ListFunctions(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(functions))
	}
}

// ---------- LIST VERSIONS ----------
func TestListFunctionVersions(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v1")
	createTestFunction(db, "calc", "v2")

	functions, err := ListFunctionVersions(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	if len(functions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(functions))
	}
}

// ---------- DEACTIVATE ----------
func TestDeactivateFunctions(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v1")

	err := DeactivateFunctions(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	fn, _ := GetFunction(db, "calc")

	if fn.Status != "inactive" {
		t.Fatalf("expected inactive, got %s", fn.Status)
	}
}

// ---------- DELETE ----------
func TestDeleteFunction(t *testing.T) {
	db := setupTestDB(t)

	createTestFunction(db, "calc", "v1")

	err := DeleteFunction(db, "calc")
	if err != nil {
		t.Fatal(err)
	}

	res, _ := GetFunction(db, "calc")

	if res != nil {
		t.Fatalf("expected nil after delete, got %+v", res)
	}
}
