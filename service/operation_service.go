package service

import (
	"asm/models"
	"asm/repository"
)

type OperationService interface {
	GetOperationSummary() (*models.OperationSummary, error)
}

type operationService struct {
	repo repository.OperationRepository
}

func NewOperationService(repo repository.OperationRepository) OperationService {
	return &operationService{repo: repo}
}

func (s *operationService) GetOperationSummary() (*models.OperationSummary, error) {
	return s.repo.GetOperationSummary()
}

