package service

import (
	"asm/models"
	"asm/repository"
)

type OperationService interface {
	GetOperationSummary(siteId string) (*models.OperationSummary, error)
	GetSites() ([]models.Site, error)
}

type operationService struct {
	repo repository.OperationRepository
}

func NewOperationService(repo repository.OperationRepository) OperationService {
	return &operationService{repo: repo}
}

func (s *operationService) GetOperationSummary(siteId string) (*models.OperationSummary, error) {
	return s.repo.GetOperationSummary(siteId)
}

func (s *operationService) GetSites() ([]models.Site, error) {
	return s.repo.GetSites()
}
