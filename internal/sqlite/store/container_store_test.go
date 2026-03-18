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

func TestGetContainerByID_NotFound(t *testing.T) {
	db := setupContainerDB(t)

	res, err := GetContainerByID(db, "unknown")
	if err != nil {
		t.Fatal(err)
	}

	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}
}

func TestGetContainersByFunction(t *testing.T) {
	db := setupContainerDB(t)

	_ = CreateContainer(db, &models.Container{ID: "c1", FunctionID: 1, Status: "free"})
	_ = CreateContainer(db, &models.Container{ID: "c2", FunctionID: 1, Status: "busy"})
	_ = CreateContainer(db, &models.Container{ID: "c3", FunctionID: 2, Status: "free"})

	list, err := GetContainersByFunction(db, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(list))
	}
}

func TestGetFreeContainer(t *testing.T) {
	db := setupContainerDB(t)

	old := time.Now().Add(-10 * time.Minute)
	new := time.Now()

	_ = CreateContainer(db, &models.Container{
		ID:         "old",
		FunctionID: 1,
		Status:     "free",
		LastUsedAt: old,
	})

	_ = CreateContainer(db, &models.Container{
		ID:         "new",
		FunctionID: 1,
		Status:     "free",
		LastUsedAt: new,
	})

	res, err := GetFreeContainer(db, 1)
	if err != nil {
		t.Fatal(err)
	}

	if res.ID != "old" {
		t.Fatalf("expected oldest container, got %s", res.ID)
	}
}

func TestGetFreeContainer_None(t *testing.T) {
	db := setupContainerDB(t)

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "busy",
	})

	res, err := GetFreeContainer(db, 1)
	if err != nil {
		t.Fatal(err)
	}

	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}
}

func TestMarkContainerBusy(t *testing.T) {
	db := setupContainerDB(t)

	_ = CreateContainer(db, &models.Container{
		ID:         "c1",
		FunctionID: 1,
		Status:     "free",
	})

	err := MarkContainerBusy(db, "c1")
	if err != nil {
		t.Fatal(err)
	}

	c, _ := GetContainerByID(db, "c1")

	if c.Status != "busy" {
		t.Fatalf("expected busy, got %s", c.Status)
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

	if c.LastUsedAt.Before(old) {
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
