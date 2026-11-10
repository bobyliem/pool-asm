package service

import (
	"asm/models"
	"asm/repository"
)

type DashboardService interface {
	GetDashboardVisitorSummary() (*models.DashboardVisitorSummary, error)
}

type dashboardService struct {
	repo repository.DashboardRepository
}

func NewDashboardService(repo repository.DashboardRepository) DashboardService {
	return &dashboardService{repo: repo}
}

func (s *dashboardService) GetDashboardVisitorSummary() (*models.DashboardVisitorSummary, error) {
	return s.repo.GetVisitorSummary()
}

