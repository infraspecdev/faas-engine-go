package store

import (
	"database/sql"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupContainerDB(t *testing.T) *sql.DB {
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

func TestCreateAndGetContainer(t *testing.T) {
	db := setupContainerDB(t)

	c := &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "free",
		HostPort:   "8080",
		LastUsedAt: time.Now(),
	}

	err := CreateContainer(db, c)
	if err != nil {
		t.Fatal(err)
	}

	res, err := GetContainerByID(db, "c1")
	if err != nil {
		t.Fatal(err)
	}

	if res == nil || res.ID != "c1" {
		t.Fatalf("expected container c1, got %+v", res)
	}
}

func TestAcquireFreeContainer(t *testing.T) {
	db := setupContainerDB(t)

	now := time.Now()

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "free",
		LastUsedAt: now.Add(-10 * time.Minute),
	})

	_ = CreateContainer(db, &models.Container{
		ID:         "c2",
		FunctionID: 1,
		Status:     "free",
		LastUsedAt: now,
	})

	c, err := AcquireFreeContainer(db, 1)
	if err != nil {
		t.Fatal(err)
	}

	if c == nil {
		t.Fatal("expected container, got nil")
	}

	if c.ID != "c2" {
		t.Fatalf("expected c2, got %s", c.ID)
	}

	updated, _ := GetContainerByID(db, c.ID)
	if updated.Status != "busy" {
		t.Fatalf("expected busy, got %s", updated.Status)
	}
}

func TestAcquireFreeContainer_None(t *testing.T) {
	db := setupContainerDB(t)

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "busy",
	})

	c, err := AcquireFreeContainer(db, 1)
	if err != nil {
		t.Fatal(err)
	}

	if c != nil {
		t.Fatalf("expected nil, got %+v", c)
	}
}

func TestMarkContainerFree(t *testing.T) {
	db := setupContainerDB(t)

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "busy",
	})

	err := MarkContainerFree(db, "c1")
	if err != nil {
		t.Fatal(err)
	}

	c, _ := GetContainerByID(db, "c1")

	if c.Status != "free" {
		t.Fatalf("expected free, got %s", c.Status)
	}
}

func TestUpdateContainerLastUsed(t *testing.T) {
	db := setupContainerDB(t)

	old := time.Now().Add(-1 * time.Hour)

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "free",
		LastUsedAt: old,
	})

	err := UpdateContainerLastUsed(db, "c1")
	if err != nil {
		t.Fatal(err)
	}

	c, _ := GetContainerByID(db, "c1")

	if !c.LastUsedAt.After(old) {
		t.Fatalf("expected updated timestamp")
	}
}

func TestCleanupIdleContainers(t *testing.T) {
	db := setupContainerDB(t)

	old := time.Now().Add(-10 * time.Minute)

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "free",
		LastUsedAt: old,
	})

	_ = CreateContainer(db, &models.Container{
		ID:         "c2",
		FunctionID: 1,
		Status:     "busy",
		LastUsedAt: old,
	})

	var deleted []string

	CleanupIdleContainers(5*time.Minute, func(id string) {
		deleted = append(deleted, id)
	})

	time.Sleep(200 * time.Millisecond)

	if len(deleted) != 1 || deleted[0] != "c1" {
		t.Fatalf("expected c1 to be cleaned, got %+v", deleted)
	}
}

func TestRemoveContainer(t *testing.T) {
	db := setupContainerDB(t)

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "free",
	})

	err := RemoveContainer(db, "c1")
	if err != nil {
		t.Fatal(err)
	}

	res, _ := GetContainerByID(db, "c1")

	if res != nil {
		t.Fatalf("expected nil after delete, got %+v", res)
	}
}
