package service

import (
	"asm/models"
	"asm/repository"
)

type CsvDashboardService interface {
	GetVisitorSummary() (*models.DashboardVisitorSummary, error)
	GetAreas() ([]string, error)
	GetCategoryDashboardData(area string) (*models.CsvDashboardDataResponse, error)
}

type csvDashboardService struct {
	repo repository.CsvDashboardRepository
}

func NewCsvDashboardService(repo repository.CsvDashboardRepository) CsvDashboardService {
	return &csvDashboardService{repo: repo}
}

func (s *csvDashboardService) GetVisitorSummary() (*models.DashboardVisitorSummary, error) {
	return s.repo.GetVisitorSummary()
}

func (s *csvDashboardService) GetAreas() ([]string, error) {
	return s.repo.GetAreas()
}

func (s *csvDashboardService) GetCategoryDashboardData(area string) (*models.CsvDashboardDataResponse, error) {
	return s.repo.GetCategoryDashboardData(area)
}
