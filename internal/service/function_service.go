package service

import (
	"faas-engine-go/internal/sqlite/models"
)

type VersionInfo struct {
	Version string `json:"version"`
	Active  bool   `json:"active"`
}

type FunctionStore interface {
	ListFunctionVersions(name string) ([]models.Function, error)
}

type FunctionVersionService struct {
	store FunctionStore
}

func NewFunctionVersionService(s FunctionStore) *FunctionVersionService {
	return &FunctionVersionService{store: s}
}

func (s *FunctionVersionService) GetVersions(name string) ([]VersionInfo, error) {

	fns, err := s.store.ListFunctionVersions(name)
	if err != nil {
		return nil, err
	}

	if len(fns) == 0 {
		return nil, ErrFunctionNotFound
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
