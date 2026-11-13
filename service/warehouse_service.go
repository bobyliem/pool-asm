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
	pivotItems, sites, err := s.repo.GetCriticalInventoryPivot(site)
	if err != nil {
		return nil, err
	}

	items, _ := s.repo.GetCriticalInventory(site)
	if items == nil {
		items = []models.CriticalInventoryItem{}
	}

	summary := models.CriticalInventorySummary{
		TotalItems: len(pivotItems),
	}

	for _, it := range pivotItems {
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
		PivotData:    pivotItems,
		LastUpdate:   lastUpdateStr,
	}

	return resp, nil
}
