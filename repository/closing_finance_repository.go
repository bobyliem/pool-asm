package repository

import (
	"asm/models"
	"strings"

	"gorm.io/gorm"
)

type ClosingFinanceRepository interface {
	GetAll(filter models.ClosingFinanceFilter) ([]models.ExtClosing, int64, models.ClosingFinanceSummary, error)
	TruncateAndInsert(items []models.ExtClosing) error
	Truncate() error
	GetFilterOptions() ([]string, []string, []string, error)
}

type closingFinanceRepository struct {
	db *gorm.DB
}

func NewClosingFinanceRepository(db *gorm.DB) ClosingFinanceRepository {
	return &closingFinanceRepository{db: db}
}

func (r *closingFinanceRepository) GetAll(filter models.ClosingFinanceFilter) ([]models.ExtClosing, int64, models.ClosingFinanceSummary, error) {
	var list []models.ExtClosing
	var totalRows int64
	var summary models.ClosingFinanceSummary

	// Base query
	tx := r.db.Model(&models.ExtClosing{})

	// Search keyword
	if s := strings.TrimSpace(filter.Search); s != "" {
		like := "%" + s + "%"
		tx = tx.Where("nopol LIKE ? OR service_order LIKE ? OR cust LIKE ? OR cust_code LIKE ? OR prd LIKE ? OR sku LIKE ? OR information LIKE ? OR notes LIKE ?",
			like, like, like, like, like, like, like, like)
	}

	// Filter Customer
	if c := strings.TrimSpace(filter.Customer); c != "" && c != "ALL" {
		tx = tx.Where("cust = ?", c)
	}

	// Filter Category
	if cat := strings.TrimSpace(filter.Category); cat != "" && cat != "ALL" {
		tx = tx.Where("category = ?", cat)
	}

	// Filter Tipe (Product / Service)
	if tp := strings.TrimSpace(filter.Tipe); tp != "" && tp != "ALL" {
		tx = tx.Where("tipe = ?", tp)
	}

	// Filter Date Range (berdasarkan completed atau released)
	if filter.TglAwal != "" {
		tx = tx.Where("DATE(released) >= ? OR DATE(completed) >= ?", filter.TglAwal, filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		tx = tx.Where("DATE(released) <= ? OR DATE(completed) <= ?", filter.TglAkhir, filter.TglAkhir)
	}

	// Hitung total matching rows
	if err := tx.Count(&totalRows).Error; err != nil {
		return nil, 0, summary, err
	}

	// Hitung summary aggregate
	type aggResult struct {
		TotalSpk      int64   `gorm:"column:total_spk"`
		TotalUnit     int64   `gorm:"column:total_unit"`
		TotalCustomer int64   `gorm:"column:total_customer"`
		GrandTotal    float64 `gorm:"column:grand_total"`
	}
	var agg aggResult
	txSummary := r.db.Model(&models.ExtClosing{})
	// Copy conditions to summary query
	if s := strings.TrimSpace(filter.Search); s != "" {
		like := "%" + s + "%"
		txSummary = txSummary.Where("nopol LIKE ? OR service_order LIKE ? OR cust LIKE ? OR cust_code LIKE ? OR prd LIKE ? OR sku LIKE ? OR information LIKE ? OR notes LIKE ?",
			like, like, like, like, like, like, like, like)
	}
	if c := strings.TrimSpace(filter.Customer); c != "" && c != "ALL" {
		txSummary = txSummary.Where("cust = ?", c)
	}
	if cat := strings.TrimSpace(filter.Category); cat != "" && cat != "ALL" {
		txSummary = txSummary.Where("category = ?", cat)
	}
	if tp := strings.TrimSpace(filter.Tipe); tp != "" && tp != "ALL" {
		txSummary = txSummary.Where("tipe = ?", tp)
	}
	if filter.TglAwal != "" {
		txSummary = txSummary.Where("DATE(released) >= ? OR DATE(completed) >= ?", filter.TglAwal, filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		txSummary = txSummary.Where("DATE(released) <= ? OR DATE(completed) <= ?", filter.TglAkhir, filter.TglAkhir)
	}

	txSummary.Select("COUNT(DISTINCT service_order) as total_spk, COUNT(DISTINCT nopol) as total_unit, COUNT(DISTINCT cust) as total_customer, COALESCE(SUM(total), 0) as grand_total").Scan(&agg)

	summary.TotalRows = totalRows
	summary.TotalSPK = agg.TotalSpk
	summary.TotalUnit = agg.TotalUnit
	summary.TotalCustomer = agg.TotalCustomer
	summary.GrandTotal = agg.GrandTotal

	// Pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Ambil data dengan sort order ID asc / released desc
	err := tx.Order("released DESC, id ASC").Limit(limit).Offset(offset).Find(&list).Error
	if err != nil {
		return nil, 0, summary, err
	}

	return list, totalRows, summary, nil
}

func (r *closingFinanceRepository) TruncateAndInsert(items []models.ExtClosing) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Truncate table
		if err := tx.Exec("TRUNCATE TABLE ext_closing").Error; err != nil {
			return err
		}

		if len(items) == 0 {
			return nil
		}

		// Insert in batches of 500
		if err := tx.CreateInBatches(items, 500).Error; err != nil {
			return err
		}

		// Update cust_code berdasarkan imp_service_order
		updateQuery := `
			UPDATE ext_closing cl
			LEFT JOIN (
				SELECT service_order, MAX(customer_code) AS customer_code
				FROM imp_service_order
				GROUP BY service_order
			) AS so ON cl.service_order = so.service_order
			SET cl.cust_code = so.customer_code
			WHERE cl.service_order = so.service_order
		`
		if err := tx.Exec(updateQuery).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *closingFinanceRepository) Truncate() error {
	return r.db.Exec("TRUNCATE TABLE ext_closing").Error
}

func (r *closingFinanceRepository) GetFilterOptions() ([]string, []string, []string, error) {
	var customers []string
	var categories []string
	var types []string

	r.db.Model(&models.ExtClosing{}).Where("cust <> ''").Distinct("cust").Order("cust ASC").Pluck("cust", &customers)
	r.db.Model(&models.ExtClosing{}).Where("category <> ''").Distinct("category").Order("category ASC").Pluck("category", &categories)
	r.db.Model(&models.ExtClosing{}).Where("tipe <> ''").Distinct("tipe").Order("tipe ASC").Pluck("tipe", &types)

	return customers, categories, types, nil
}
