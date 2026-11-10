package repository

import (
	"asm/models"
	"strings"

	"gorm.io/gorm"
)

type PriorUnitRepository interface {
	GetAll(kategori, statue, search string) ([]models.PriorUnit, models.PriorUnitSummary, error)
	GetByID(id int) (*models.PriorUnit, error)
	Create(unit *models.PriorUnit) error
	Update(unit *models.PriorUnit) error
	Delete(id int) error
}

type priorUnitRepository struct {
	db *gorm.DB
}

func NewPriorUnitRepository(db *gorm.DB) PriorUnitRepository {
	return &priorUnitRepository{db: db}
}

func (r *priorUnitRepository) GetAll(kategori, statue, search string) ([]models.PriorUnit, models.PriorUnitSummary, error) {
	var list []models.PriorUnit
	var summary models.PriorUnitSummary

	// Hitung summary global
	var totalCount int64
	var openCount int64
	var doneCount int64

	r.db.Model(&models.PriorUnit{}).Count(&totalCount)
	r.db.Model(&models.PriorUnit{}).Where("statue = ?", "0").Count(&openCount)
	r.db.Model(&models.PriorUnit{}).Where("statue = ?", "1").Count(&doneCount)

	summary.Total = int(totalCount)
	summary.Open = int(openCount)
	summary.Done = int(doneCount)

	// Query data dengan filter
	tx := r.db.Model(&models.PriorUnit{})

	if kategori != "" && kategori != "ALL" {
		tx = tx.Where("LOWER(kategori) = ?", strings.ToLower(kategori))
	}

	if statue != "" && statue != "ALL" {
		tx = tx.Where("statue = ?", statue)
	}

	if search != "" {
		s := "%" + strings.TrimSpace(search) + "%"
		tx = tx.Where("nopol LIKE ? OR cabin LIKE ? OR jenis LIKE ? OR note LIKE ?", s, s, s, s)
	}

	// Order by: yang Open ("0") di atas, lalu order by id DESC
	if err := tx.Order("statue ASC, id DESC").Find(&list).Error; err != nil {
		return nil, summary, err
	}

	return list, summary, nil
}

func (r *priorUnitRepository) GetByID(id int) (*models.PriorUnit, error) {
	var unit models.PriorUnit
	if err := r.db.First(&unit, id).Error; err != nil {
		return nil, err
	}
	return &unit, nil
}

func (r *priorUnitRepository) Create(unit *models.PriorUnit) error {
	return r.db.Create(unit).Error
}

func (r *priorUnitRepository) Update(unit *models.PriorUnit) error {
	return r.db.Save(unit).Error
}

func (r *priorUnitRepository) Delete(id int) error {
	return r.db.Delete(&models.PriorUnit{}, id).Error
}
