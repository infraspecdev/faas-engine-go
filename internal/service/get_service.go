package service

import "faas-engine-go/internal/sqlite/models"

type FunctionLister interface {
	ListFunctions() ([]models.Function, error)
}

type functionListService struct {
	store Store
}

func NewListService(s Store) FunctionLister {
	return &functionListService{store: s}
}

func (s *functionListService) ListFunctions() ([]models.Function, error) {
	return s.store.ListFunctions()
}
