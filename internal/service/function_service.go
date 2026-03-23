package service

import (
	"fmt"

	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/store"
)

type VersionInfo struct {
	Version string `json:"version"`
	Active  bool   `json:"active"`
}

func GetFunctionVersions(name string) ([]VersionInfo, error) {

	fns, err := store.ListFunctionVersions(sqlite.DB, name)
	if err != nil {
		return nil, err
	}

	if len(fns) == 0 {
		return nil, fmt.Errorf("function not found")
	}

	var result []VersionInfo

	for _, fn := range fns {
		result = append(result, VersionInfo{
			Version: fn.Version,
			Active:  fn.Status == "active",
		})
	}

	return result, nil
}
