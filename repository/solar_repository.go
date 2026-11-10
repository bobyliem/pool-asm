package repository

import (
	"strings"
	"time"

	"asm/models"

	"gorm.io/gorm"
)

type SolarRepository interface {
	GetPemakaianSolar(filter models.PemakaianSolarFilter) ([]models.PemakaianSolar, int64, error)
	GetSummary(filter models.PemakaianSolarFilter) (models.SolarSummary, error)
	GetFilterOptions() ([]string, []string, string, string, error)
}

type solarRepository struct {
	db *gorm.DB
}

func NewSolarRepository(db *gorm.DB) SolarRepository {
	return &solarRepository{db: db}
}

func (r *solarRepository) buildQuery(filter models.PemakaianSolarFilter) *gorm.DB {
	q := r.db.Model(&models.PemakaianSolar{})

	if filter.TglAwal != "" {
		q = q.Where("tanggal >= ?", filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		q = q.Where("tanggal <= ?", filter.TglAkhir)
	}

	cleanUnit := strings.TrimSpace(filter.Unit)
	if cleanUnit != "" && !strings.EqualFold(cleanUnit, "all") && !strings.EqualFold(cleanUnit, "Semua Unit") {
		q = q.Where("unit = ?", cleanUnit)
	}

	cleanShift := strings.TrimSpace(filter.Shift)
	if cleanShift != "" && !strings.EqualFold(cleanShift, "all") && !strings.EqualFold(cleanShift, "Semua Shift") {
		q = q.Where("shift = ?", cleanShift)
	}

	return q
}

func (r *solarRepository) GetPemakaianSolar(filter models.PemakaianSolarFilter) ([]models.PemakaianSolar, int64, error) {
	var results []models.PemakaianSolar
	var totalRows int64

	q := r.buildQuery(filter)

	if err := q.Count(&totalRows).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 1000 {
		limit = 50
	}
	offset := (page - 1) * limit

	err := q.Order("tanggal DESC, unit ASC, shift ASC").
		Limit(limit).
		Offset(offset).
		Find(&results).Error

	return results, totalRows, err
}

func (r *solarRepository) GetSummary(filter models.PemakaianSolarFilter) (models.SolarSummary, error) {
	var summary models.SolarSummary
	q := r.buildQuery(filter)

	type AggResult struct {
		TotalRows       int64   `gorm:"column:total_rows"`
		TotalPakaiLiter float64 `gorm:"column:total_pakai_liter"`
		TotalIsiLiter   float64 `gorm:"column:total_isi_liter"`
		TotalJamHm     float64 `gorm:"column:total_jam_hm"`
	}

	var agg AggResult
	err := q.Select("COUNT(*) as total_rows, COALESCE(SUM(pakai_liter), 0) as total_pakai_liter, COALESCE(SUM(isi_liter), 0) as total_isi_liter, COALESCE(SUM(total_jam_hm), 0) as total_jam_hm").
		Scan(&agg).Error

	if err != nil {
		return summary, err
	}

	summary.TotalRows = agg.TotalRows
	summary.TotalPakaiLiter = agg.TotalPakaiLiter
	summary.TotalIsiLiter = agg.TotalIsiLiter
	summary.TotalJamHm = agg.TotalJamHm

	if agg.TotalJamHm > 0 {
		summary.AvgLiterPerHm = agg.TotalPakaiLiter / agg.TotalJamHm
	} else {
		summary.AvgLiterPerHm = 0
	}

	return summary, nil
}

func (r *solarRepository) GetFilterOptions() ([]string, []string, string, string, error) {
	var units []string
	var minDate *time.Time
	var maxDate *time.Time

	// Unique units
	r.db.Model(&models.PemakaianSolar{}).
		Distinct("unit").
		Order("unit ASC").
		Pluck("unit", &units)

	// Master units terdaftar
	masterUnits := []string{
		"SK330-8", "SK330-8 02", "SK330-8 03", "SK330-8 04", "SK330-8 05", "SK330-8 06",
		"SK200-10 05", "CAT 966G", "CAT 950G", "CAT 950G 02", "KOB 03", "PC200-8", "KOB 04",
	}

	unitMap := make(map[string]bool)
	var finalUnits []string
	for _, u := range masterUnits {
		if !unitMap[u] {
			unitMap[u] = true
			finalUnits = append(finalUnits, u)
		}
	}
	for _, u := range units {
		cleanU := strings.TrimSpace(u)
		if cleanU != "" && !unitMap[cleanU] {
			unitMap[cleanU] = true
			finalUnits = append(finalUnits, cleanU)
		}
	}

	shifts := []string{"Pagi", "Malam"}

	// Min and Max Tanggal
	r.db.Model(&models.PemakaianSolar{}).Select("MIN(tanggal)").Scan(&minDate)
	r.db.Model(&models.PemakaianSolar{}).Select("MAX(tanggal)").Scan(&maxDate)

	minTglStr := ""
	maxTglStr := ""
	if minDate != nil {
		minTglStr = minDate.Format("2006-01-02")
	}
	if maxDate != nil {
		maxTglStr = maxDate.Format("2006-01-02")
	}

	return finalUnits, shifts, minTglStr, maxTglStr, nil
}
