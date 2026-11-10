package service

import (
	"fmt"
	"time"

	"asm/models"
	"asm/repository"
)

type WarehouseService interface {
	GetCriticalInventory(site string) (*models.CriticalInventoryResponse, error)
	GetSites() ([]string, error)
}

type warehouseService struct {
	repo repository.WarehouseRepository
}

func NewWarehouseService(repo repository.WarehouseRepository) WarehouseService {
	return &warehouseService{repo: repo}
}

func (s *warehouseService) GetSites() ([]string, error) {
	return s.repo.GetSites()
}

func (s *warehouseService) GetCriticalInventory(site string) (*models.CriticalInventoryResponse, error) {
	items, err := s.repo.GetCriticalInventory(site)
	if err != nil {
		return nil, err
	}

	sites, err := s.repo.GetSites()
	if err != nil {
		sites = []string{}
	}

	summary := models.CriticalInventorySummary{
		TotalItems: len(items),
	}

	for _, it := range items {
		switch it.Note {
		case "UrgentReStock":
			summary.UrgentRestockCount++
		case "LowStock":
			summary.LowStockCount++
		case "Check":
			summary.CheckCount++
		}

		if it.QtyPo != nil {
			summary.TotalQtyPoPending += *it.QtyPo
		}
	}

	now := time.Now()
	monthHeaders := models.GetMonthHeadersHelper(now)

	indonesianMonths := []string{
		"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des",
	}
	lastUpdateStr := fmt.Sprintf("%02d %s %d %02d:%02d:%02d WIB",
		now.Day(), indonesianMonths[int(now.Month())], now.Year(),
		now.Hour(), now.Minute(), now.Second())

	resp := &models.CriticalInventoryResponse{
		Status:       "success",
		SelectedSite: site,
		Summary:      summary,
		MonthHeaders: monthHeaders,
		Sites:        sites,
		Data:         items,
		LastUpdate:   lastUpdateStr,
	}

	return resp, nil
}
