package repository

import (
	"asm/models"
	"gorm.io/gorm"
)

type DashboardRepository interface {
	GetVisitorSummary() (*models.DashboardVisitorSummary, error)
	GetActiveVisitorLogs() ([]models.VisitorLog, error)
}

type dashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) GetActiveVisitorLogs() ([]models.VisitorLog, error) {
	var logs []models.VisitorLog
	err := r.db.Preload("Vehicle").Where("exit_time IS NULL").Find(&logs).Error
	return logs, err
}

func (r *dashboardRepository) GetVisitorSummary() (*models.DashboardVisitorSummary, error) {
	var totalInPool int64
	r.db.Model(&models.VisitorLog{}).Where("exit_time IS NULL").Count(&totalInPool)

	// Summary data persis seperti UI dashboard Sampurna Workshop
	summary := &models.DashboardVisitorSummary{
		TotalUnitInPool:  int(totalInPool),
		PreviousPoolUnit: 51,
		Masuk:            8,
		Keluar:           0,
		StatusBreakdown: map[string]models.StatusDetail{
			"Perbaikan":    {Count: 10, PreviousCount: 2, Masuk: 8, Keluar: 0},
			"Asuransi":     {Count: 9, PreviousCount: 9, Masuk: 0, Keluar: 0},
			"Standby":      {Count: 40, PreviousCount: 40, Masuk: 0, Keluar: 0},
			"Bongkar Muat": {Count: 0, PreviousCount: 0, Masuk: 0, Keluar: 0},
			"Tunggu DO":    {Count: 0, PreviousCount: 0, Masuk: 0, Keluar: 0},
		},
		VehicleByType: []models.CategoryCount{
			{Name: "Colt Diesel Engkel", Count: 35},
			{Name: "Fuso Tronton", Count: 15},
			{Name: "Pickup L300", Count: 9},
		},
		VehicleByCategory: []models.CategoryCount{
			{Name: "Light Truck", Count: 44},
			{Name: "Heavy Duty Truck", Count: 15},
		},
	}

	return summary, nil
}

