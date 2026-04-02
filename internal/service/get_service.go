package service

import (
	"faas-engine-go/internal/core"
	"faas-engine-go/internal/sqlite/models"
)

type FunctionLister interface {
	ListFunctions() ([]models.Function, error)
}

type functionListService struct {
	store core.Store
}

func NewListService(s core.Store) FunctionLister {
	return &functionListService{store: s}
}

func (s *functionListService) ListFunctions() ([]models.Function, error) {
	return s.store.ListFunctions()
}
