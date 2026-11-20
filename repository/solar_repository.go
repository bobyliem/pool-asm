package repository

import (
	"fmt"
	"strings"
	"time"

	"asm/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SolarRepository interface {
	GetPemakaianSolar(filter models.PemakaianSolarFilter) ([]models.PemakaianSolar, int64, error)
	GetSummary(filter models.PemakaianSolarFilter) (models.SolarSummary, error)
	GetFilterOptions() ([]string, []string, string, string, error)

	// DtSolar Methods
	GetDtSolar(filter models.DtSolarFilter) ([]models.DtSolar, int64, error)
	GetDtSolarSummary(filter models.DtSolarFilter) (models.DtSolarSummary, error)
	GetDtSolarFilterOptions() ([]string, []int, string, string, error)
	UpsertBatchDtSolar(records []models.DtSolar) (int, error)
	TruncateDtSolar() error
	PostImportSync() error

	// Laporan Colokan Solar
	GetLaporanDetailHarian(filter models.LaporanDetailHarianFilter) ([]models.LaporanDetailHarianItem, int64, models.LaporanDetailHarianSummary, error)
	GetLaporanRekapUnit(filter models.LaporanRekapUnitFilter) ([]models.LaporanRekapUnitItem, models.LaporanRekapUnitSummary, error)
	GetLaporanRekonsiliasi(filter models.LaporanRekonsiliasiFilter) ([]models.LaporanRekonsiliasiItem, int64, models.LaporanRekonsiliasiSummary, error)
	GetDetailPoints(tgl string, unit string) ([]models.DtSolar, error)
	GetLaporanFilterOptions() ([]string, []string, string, string, error)
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

// ==========================================
// DT_SOLAR IMPLEMENTATION
// ==========================================

func (r *solarRepository) buildDtSolarQuery(filter models.DtSolarFilter) *gorm.DB {
	q := r.db.Model(&models.DtSolar{})

	if filter.TglAwal != "" {
		q = q.Where("tgl >= ?", filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		q = q.Where("tgl <= ?", filter.TglAkhir)
	}

	cleanUnit := strings.TrimSpace(filter.Unit)
	if cleanUnit != "" && !strings.EqualFold(cleanUnit, "all") && !strings.EqualFold(cleanUnit, "Semua Unit") {
		q = q.Where("unit = ?", cleanUnit)
	}

	cleanJam := strings.TrimSpace(filter.Jam)
	if cleanJam != "" && !strings.EqualFold(cleanJam, "all") && !strings.EqualFold(cleanJam, "Semua Jam") {
		q = q.Where("jam = ?", cleanJam)
	}

	return q
}

func (r *solarRepository) GetDtSolar(filter models.DtSolarFilter) ([]models.DtSolar, int64, error) {
	var results []models.DtSolar
	var totalRows int64

	q := r.buildDtSolarQuery(filter)

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

	err := q.Order("tgl DESC, unit ASC, jam ASC").
		Limit(limit).
		Offset(offset).
		Find(&results).Error

	return results, totalRows, err
}

func (r *solarRepository) GetDtSolarSummary(filter models.DtSolarFilter) (models.DtSolarSummary, error) {
	var summary models.DtSolarSummary
	q := r.buildDtSolarQuery(filter)

	type AggResult struct {
		TotalRows int64   `gorm:"column:total_rows"`
		TotalIsi  float64 `gorm:"column:total_isi"`
		TotalHm   float64 `gorm:"column:total_hm"`
		TotalUnit int64   `gorm:"column:total_unit"`
	}

	var agg AggResult
	err := q.Select("COUNT(*) as total_rows, COALESCE(SUM(isi), 0) as total_isi, COALESCE(SUM(hm), 0) as total_hm, COUNT(DISTINCT unit) as total_unit").
		Scan(&agg).Error

	if err != nil {
		return summary, err
	}

	summary.TotalRows = agg.TotalRows
	summary.TotalIsi = agg.TotalIsi
	summary.TotalHm = agg.TotalHm
	summary.TotalUnit = agg.TotalUnit

	return summary, nil
}

func (r *solarRepository) GetDtSolarFilterOptions() ([]string, []int, string, string, error) {
	var units []string
	var jams []int

	// Unique units
	r.db.Model(&models.DtSolar{}).
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

	// Distinct jam in DB or default list
	r.db.Model(&models.DtSolar{}).
		Distinct("jam").
		Order("jam ASC").
		Pluck("jam", &jams)

	if len(jams) == 0 {
		jams = []int{7, 16, 17, 18, 22, 23}
	}

	// Min and Max Tanggal (Gunakan DATE_FORMAT agar aman dari Scan error NULL / pointer type)
	var minTglStr, maxTglStr string
	_ = r.db.Raw("SELECT COALESCE(DATE_FORMAT(MIN(tgl), '%Y-%m-%d'), ''), COALESCE(DATE_FORMAT(MAX(tgl), '%Y-%m-%d'), '') FROM dt_solar").Row().Scan(&minTglStr, &maxTglStr)

	return finalUnits, jams, minTglStr, maxTglStr, nil
}

func (r *solarRepository) UpsertBatchDtSolar(records []models.DtSolar) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	// Upsert using GORM Clause OnConflict with batching
	err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tgl"},
			{Name: "unit"},
			{Name: "jam"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"hm", "awal", "akhir", "isi", "satuan"}),
	}).CreateInBatches(records, 200).Error

	if err != nil {
		return 0, fmt.Errorf("gagal upsert batch dt_solar: %w", err)
	}

	return len(records), nil
}

func (r *solarRepository) TruncateDtSolar() error {
	return r.db.Exec("TRUNCATE TABLE dt_solar").Error
}

// PostImportSync executes Query 1 and Query 2 after import data
func (r *solarRepository) PostImportSync() error {
	// Query 1: TRUNCATE dt_solar_his and copy from dt_solar
	if err := r.db.Exec("TRUNCATE dt_solar_his").Error; err != nil {
		return fmt.Errorf("gagal truncate dt_solar_his: %w", err)
	}

	insertSQL := `INSERT INTO dt_solar_his(tgl, unit, jam, hm, awal, akhir, isi, satuan)
SELECT tgl, unit, jam, hm, awal, akhir, isi, satuan
FROM dt_solar
ORDER BY tgl, unit, jam`
	if err := r.db.Exec(insertSQL).Error; err != nil {
		return fmt.Errorf("gagal copy data ke dt_solar_his: %w", err)
	}

	// Query 2: CALL sp_solar_sum(CURDATE(), 0)
	if err := r.db.Exec("CALL sp_solar_sum(CURDATE(), 0)").Error; err != nil {
		return fmt.Errorf("gagal memanggil sp_solar_sum: %w", err)
	}

	return nil
}

// ==========================================
// LAPORAN COLOKAN SOLAR IMPLEMENTATIONS
// ==========================================

// GetLaporanDetailHarian retrieves daily detailed records with cm to liter conversion
func (r *solarRepository) GetLaporanDetailHarian(filter models.LaporanDetailHarianFilter) ([]models.LaporanDetailHarianItem, int64, models.LaporanDetailHarianSummary, error) {
	var items []models.LaporanDetailHarianItem
	var summary models.LaporanDetailHarianSummary

	baseWhere := "WHERE 1=1"
	var args []interface{}

	if filter.TglAwal != "" {
		baseWhere += " AND s.tgl >= ?"
		args = append(args, filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		baseWhere += " AND s.tgl <= ?"
		args = append(args, filter.TglAkhir)
	}
	cleanArea := strings.TrimSpace(filter.Area)
	if cleanArea != "" && !strings.EqualFold(cleanArea, "all") && !strings.EqualFold(cleanArea, "Semua Area") {
		baseWhere += " AND u.area = ?"
		args = append(args, cleanArea)
	}
	cleanUnit := strings.TrimSpace(filter.Unit)
	if cleanUnit != "" && !strings.EqualFold(cleanUnit, "all") && !strings.EqualFold(cleanUnit, "Semua Unit") {
		baseWhere += " AND s.unit = ?"
		args = append(args, cleanUnit)
	}
	if filter.Search != "" {
		searchLike := "%" + strings.TrimSpace(filter.Search) + "%"
		baseWhere += " AND (s.unit LIKE ? OR DATE_FORMAT(s.tgl, '%Y-%m-%d') LIKE ?)"
		args = append(args, searchLike, searchLike)
	}

	// 1. Total count and summary
	summarySQL := fmt.Sprintf(`
SELECT 
    COUNT(*) AS total_records,
    ROUND(COALESCE(SUM(s.hm_akhir - s.hm_awal), 0), 2) AS total_hm,
    ROUND(COALESCE(SUM(s.pakai), 0), 2) AS total_pakai_liter,
    ROUND(COALESCE(SUM(s.isi), 0), 2) AS total_isi_liter,
    COUNT(DISTINCT s.unit) AS total_unit,
    CASE 
        WHEN COALESCE(SUM(s.hm_akhir - s.hm_awal), 0) > 0 
        THEN ROUND(COALESCE(SUM(s.pakai), 0) / SUM(s.hm_akhir - s.hm_awal), 2)
        ELSE 0
    END AS avg_ltr_per_hm,
    ROUND(COALESCE(SUM(s.isi), 0) - COALESCE(SUM(s.pakai), 0), 2) AS netto_liter
FROM dt_solar_summary s
LEFT JOIN dt_solar_unit u ON s.unit = u.unit
%s
`, baseWhere)

	if err := r.db.Raw(summarySQL, args...).Scan(&summary).Error; err != nil {
		return nil, 0, summary, fmt.Errorf("gagal menghitung summary detail harian: %w", err)
	}

	// 2. Paginated rows
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	querySQL := fmt.Sprintf(`
SELECT 
    s.id,
    DATE_FORMAT(s.tgl, '%%Y-%%m-%%d') AS tgl,
    COALESCE(u.area, '-') AS area,
    s.unit,
    s.hm_awal,
    s.hm_akhir,
    ROUND(s.hm_akhir - s.hm_awal, 2) AS total_hm,
    COALESCE(s.saldoawal, 0) AS saldo_awal,
    ROUND(s.pakai, 2) AS pakai_liter,
    s.isi AS isi_liter,
    COALESCE(s.saldoakhir, 0) AS saldo_akhir,
    CASE 
        WHEN (s.hm_akhir - s.hm_awal) > 0 THEN ROUND(s.pakai / (s.hm_akhir - s.hm_awal), 2)
        ELSE 0
    END AS ltr_per_hm,
    ROUND(s.isi - s.pakai, 2) AS netto_liter,
    COALESCE(DATE_FORMAT(s.created, '%%Y-%%m-%%d %%H:%%i:%%s'), '-') AS created
FROM dt_solar_summary s
LEFT JOIN dt_solar_unit u ON s.unit = u.unit
%s
ORDER BY s.tgl DESC, s.unit ASC
LIMIT ? OFFSET ?
`, baseWhere)

	queryArgs := append(args, limit, offset)
	if err := r.db.Raw(querySQL, queryArgs...).Scan(&items).Error; err != nil {
		return nil, summary.TotalRecords, summary, fmt.Errorf("gagal mengambil data detail harian: %w", err)
	}

	return items, summary.TotalRecords, summary, nil
}

// GetLaporanRekapUnit retrieves aggregated summary per unit or per month/unit
func (r *solarRepository) GetLaporanRekapUnit(filter models.LaporanRekapUnitFilter) ([]models.LaporanRekapUnitItem, models.LaporanRekapUnitSummary, error) {
	var items []models.LaporanRekapUnitItem
	var summary models.LaporanRekapUnitSummary

	baseWhere := "WHERE 1=1"
	var args []interface{}

	if filter.TglAwal != "" {
		baseWhere += " AND s.tgl >= ?"
		args = append(args, filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		baseWhere += " AND s.tgl <= ?"
		args = append(args, filter.TglAkhir)
	}
	cleanArea := strings.TrimSpace(filter.Area)
	if cleanArea != "" && !strings.EqualFold(cleanArea, "all") && !strings.EqualFold(cleanArea, "Semua Area") {
		baseWhere += " AND u.area = ?"
		args = append(args, cleanArea)
	}
	cleanUnit := strings.TrimSpace(filter.Unit)
	if cleanUnit != "" && !strings.EqualFold(cleanUnit, "all") && !strings.EqualFold(cleanUnit, "Semua Unit") {
		baseWhere += " AND s.unit = ?"
		args = append(args, cleanUnit)
	}

	statusOrderCase := `
    CASE 
        WHEN SUM(s.hm_akhir - s.hm_awal) <= 0 THEN 4
        WHEN (CASE WHEN SUM(s.hm_akhir - s.hm_awal) > 0 THEN SUM(s.pakai) / SUM(s.hm_akhir - s.hm_awal) ELSE 0 END) > 20 
             AND (CASE WHEN SUM(s.hm_akhir - s.hm_awal) > 0 THEN SUM(s.pakai) / SUM(s.hm_akhir - s.hm_awal) ELSE 0 END) <= 40 THEN 1
        WHEN (CASE WHEN SUM(s.hm_akhir - s.hm_awal) > 0 THEN SUM(s.pakai) / SUM(s.hm_akhir - s.hm_awal) ELSE 0 END) > 40 THEN 2
        ELSE 3
    END ASC`

	var groupByClause, periodSelect, orderByClause string
	if filter.GroupBy == "month_unit" {
		periodSelect = "DATE_FORMAT(s.tgl, '%Y-%m') AS period"
		groupByClause = "GROUP BY DATE_FORMAT(s.tgl, '%Y-%m'), s.unit, u.area"
		orderByClause = fmt.Sprintf("ORDER BY period DESC, COALESCE(u.area, '-') ASC, %s, s.unit ASC", statusOrderCase)
	} else {
		periodSelect = "'Semua Periode' AS period"
		groupByClause = "GROUP BY s.unit, u.area"
		orderByClause = fmt.Sprintf("ORDER BY COALESCE(u.area, '-') ASC, %s, s.unit ASC", statusOrderCase)
	}

	querySQL := fmt.Sprintf(`
SELECT 
    %s,
    COALESCE(u.area, '-') AS area,
    s.unit,
    COUNT(DISTINCT s.tgl) AS hari_kerja,
    MIN(s.hm_awal) AS min_hm_awal,
    MAX(s.hm_akhir) AS max_hm_akhir,
    ROUND(SUM(s.hm_akhir - s.hm_awal), 2) AS total_hm,
    ROUND(SUM(s.hm_akhir - s.hm_awal) / COUNT(DISTINCT s.tgl), 2) AS avg_hm_per_hari,
    ROUND(SUM(s.pakai), 2) AS total_pakai_liter,
    ROUND(SUM(s.isi), 2) AS total_isi_liter,
    ROUND(SUM(s.isi) - SUM(s.pakai), 2) AS netto_liter,
    ROUND(SUM(s.pakai) / COUNT(DISTINCT s.tgl), 2) AS avg_pakai_per_hari,
    CASE 
        WHEN SUM(s.hm_akhir - s.hm_awal) > 0 THEN ROUND(SUM(s.pakai) / SUM(s.hm_akhir - s.hm_awal), 2)
        ELSE 0
    END AS ltr_per_hm,
    CASE
        WHEN SUM(s.hm_akhir - s.hm_awal) <= 0 THEN 'Idle'
        WHEN (CASE WHEN SUM(s.hm_akhir - s.hm_awal) > 0 THEN SUM(s.pakai) / SUM(s.hm_akhir - s.hm_awal) ELSE 0 END) <= 20 THEN 'Efisien'
        WHEN (CASE WHEN SUM(s.hm_akhir - s.hm_awal) > 0 THEN SUM(s.pakai) / SUM(s.hm_akhir - s.hm_awal) ELSE 0 END) <= 40 THEN 'Moderat'
        ELSE 'Tinggi'
    END AS efisiensi_status
FROM dt_solar_summary s
LEFT JOIN dt_solar_unit u ON s.unit = u.unit
%s
%s
%s
`, periodSelect, baseWhere, groupByClause, orderByClause)

	if err := r.db.Raw(querySQL, args...).Scan(&items).Error; err != nil {
		return nil, summary, fmt.Errorf("gagal mengambil data rekap unit: %w", err)
	}

	// Calculate overall summary
	summarySQL := fmt.Sprintf(`
SELECT 
    COUNT(DISTINCT s.unit) AS total_unit,
    COUNT(DISTINCT s.tgl) AS total_hari_kerja,
    ROUND(COALESCE(SUM(s.hm_akhir - s.hm_awal), 0), 2) AS total_hm,
    ROUND(COALESCE(SUM(s.pakai), 0), 2) AS total_pakai_liter,
    ROUND(COALESCE(SUM(s.isi), 0), 2) AS total_isi_liter,
    CASE 
        WHEN COALESCE(SUM(s.hm_akhir - s.hm_awal), 0) > 0 
        THEN ROUND(COALESCE(SUM(s.pakai), 0) / SUM(s.hm_akhir - s.hm_awal), 2)
        ELSE 0
    END AS avg_ltr_per_hm,
    ROUND(COALESCE(SUM(s.isi), 0) - COALESCE(SUM(s.pakai), 0), 2) AS netto_liter
FROM dt_solar_summary s
LEFT JOIN dt_solar_unit u ON s.unit = u.unit
%s
`, baseWhere)

	if err := r.db.Raw(summarySQL, args...).Scan(&summary).Error; err != nil {
		return nil, summary, fmt.Errorf("gagal menghitung summary rekap unit: %w", err)
	}

	return items, summary, nil
}

// GetLaporanRekonsiliasi retrieves reconciliation and anomaly audit records
func (r *solarRepository) GetLaporanRekonsiliasi(filter models.LaporanRekonsiliasiFilter) ([]models.LaporanRekonsiliasiItem, int64, models.LaporanRekonsiliasiSummary, error) {
	var items []models.LaporanRekonsiliasiItem
	var summary models.LaporanRekonsiliasiSummary

	baseWhere := "WHERE 1=1"
	var args []interface{}

	if filter.TglAwal != "" {
		baseWhere += " AND s.tgl >= ?"
		args = append(args, filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		baseWhere += " AND s.tgl <= ?"
		args = append(args, filter.TglAkhir)
	}
	cleanArea := strings.TrimSpace(filter.Area)
	if cleanArea != "" && !strings.EqualFold(cleanArea, "all") && !strings.EqualFold(cleanArea, "Semua Area") {
		baseWhere += " AND u.area = ?"
		args = append(args, cleanArea)
	}
	cleanUnit := strings.TrimSpace(filter.Unit)
	if cleanUnit != "" && !strings.EqualFold(cleanUnit, "all") && !strings.EqualFold(cleanUnit, "Semua Unit") {
		baseWhere += " AND s.unit = ?"
		args = append(args, cleanUnit)
	}

	rawQuery := fmt.Sprintf(`
SELECT 
    s.id,
    DATE_FORMAT(s.tgl, '%%Y-%%m-%%d') AS tgl,
    COALESCE(u.area, '-') AS area,
    s.unit,
    s.hm_awal,
    s.hm_akhir,
    ROUND(s.hm_akhir - s.hm_awal, 2) AS total_hm,
    ROUND(s.pakai, 2) AS pakai_liter,
    s.isi AS isi_liter,
    CASE 
        WHEN (s.hm_akhir - s.hm_awal) > 0 THEN ROUND(s.pakai / (s.hm_akhir - s.hm_awal), 2)
        ELSE 0
    END AS ltr_per_hm,
    ROUND(s.isi - s.pakai, 2) AS netto_liter,
    CASE
        WHEN s.hm_akhir < s.hm_awal THEN 'ANOMALI'
        WHEN (s.hm_akhir - s.hm_awal) > 24 THEN 'ANOMALI'
        WHEN (s.hm_akhir - s.hm_awal) = 0 AND s.pakai > 0 THEN 'ANOMALI'
        WHEN s.pakai < 0 THEN 'ANOMALI'
        WHEN (s.hm_akhir - s.hm_awal) > 0 AND s.pakai / (s.hm_akhir - s.hm_awal) > 50 THEN 'ANOMALI'
        WHEN (s.hm_akhir - s.hm_awal) > 0 AND s.pakai / (s.hm_akhir - s.hm_awal) > 35 THEN 'PERINGATAN'
        WHEN s.pakai > 300 AND s.isi = 0 THEN 'PERINGATAN'
        ELSE 'NORMAL'
    END AS status_audit,
    CASE
        WHEN s.hm_akhir < s.hm_awal THEN 'HM Mundur (Akhir < Awal)'
        WHEN (s.hm_akhir - s.hm_awal) > 24 THEN 'Jam Kerja > 24 Jam/Hari'
        WHEN (s.hm_akhir - s.hm_awal) = 0 AND s.pakai > 0 THEN 'Pemakaian Solar Tanpa Jam Kerja (HM = 0)'
        WHEN s.pakai < 0 THEN 'Pemakaian Negatif'
        WHEN (s.hm_akhir - s.hm_awal) > 0 AND s.pakai / (s.hm_akhir - s.hm_awal) > 50 THEN 'Konsumsi Ekstrem (>50 L/HM)'
        WHEN (s.hm_akhir - s.hm_awal) > 0 AND s.pakai / (s.hm_akhir - s.hm_awal) > 35 THEN 'Konsumsi Tinggi (>35 L/HM)'
        WHEN s.pakai > 300 AND s.isi = 0 THEN 'Konsumsi Besar Tanpa Pengisian Solar'
        ELSE 'Operasional Normal'
    END AS jenis_anomali,
    CASE
        WHEN s.hm_akhir < s.hm_awal THEN 'Indikasi kesalahan catat odometer HM atau pergantian unit meteran'
        WHEN (s.hm_akhir - s.hm_awal) > 24 THEN 'Total jam kerja harian melebihi 24 jam fisik, periksa input HM awal/akhir'
        WHEN (s.hm_akhir - s.hm_awal) = 0 AND s.pakai > 0 THEN 'Ada konsumsi solar tercatat namun HM tidak bergerak (indikasi unit idle lama atau pencurian)'
        WHEN s.pakai < 0 THEN 'Volume pemakaian solar bernilai negatif'
        WHEN (s.hm_akhir - s.hm_awal) > 0 AND s.pakai / (s.hm_akhir - s.hm_awal) > 50 THEN 'Rasio konsumsi solar per jam kerja sangat tinggi di luar batas toleransi normal armada'
        WHEN (s.hm_akhir - s.hm_awal) > 0 AND s.pakai / (s.hm_akhir - s.hm_awal) > 35 THEN 'Rasio konsumsi solar per jam di atas rata-rata operasional, perlu crosscheck beban kerja'
        WHEN s.pakai > 300 AND s.isi = 0 THEN 'Pemakaian solar harian besar tanpa adanya pengisian solar pada hari tersebut'
        ELSE 'Data konsumsi dan jam kerja berada dalam rentang toleransi normal'
    END AS keterangan_audit
FROM dt_solar_summary s
LEFT JOIN dt_solar_unit u ON s.unit = u.unit
%s
`, baseWhere)

	// Summary stats over all matching rows
	summarySQL := fmt.Sprintf(`
SELECT 
    COUNT(*) AS total_audit_rows,
    SUM(CASE WHEN t.status_audit = 'ANOMALI' THEN 1 ELSE 0 END) AS total_anomali_rows,
    SUM(CASE WHEN t.status_audit = 'PERINGATAN' THEN 1 ELSE 0 END) AS total_peringatan_rows,
    SUM(CASE WHEN t.status_audit = 'NORMAL' THEN 1 ELSE 0 END) AS total_normal_rows,
    ROUND(SUM(CASE WHEN t.status_audit IN ('ANOMALI', 'PERINGATAN') THEN t.pakai_liter ELSE 0 END), 2) AS total_pakai_anomali_liter,
    ROUND(SUM(t.netto_liter), 2) AS total_selisih_netto_liter
FROM (%s) AS t
`, rawQuery)

	if err := r.db.Raw(summarySQL, args...).Scan(&summary).Error; err != nil {
		return nil, 0, summary, fmt.Errorf("gagal menghitung summary rekonsiliasi: %w", err)
	}

	// Filter outer query for anomalies if requested
	outerWhere := "WHERE 1=1"
	var outerArgs []interface{}
	if filter.OnlyAnomali {
		outerWhere += " AND t.status_audit IN ('ANOMALI', 'PERINGATAN')"
	}
	if filter.KategoriAnomali != "" && !strings.EqualFold(filter.KategoriAnomali, "all") {
		outerWhere += " AND t.jenis_anomali LIKE ?"
		outerArgs = append(outerArgs, "%"+filter.KategoriAnomali+"%")
	}

	// Count total filtered rows
	var filteredTotal int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS t %s", rawQuery, outerWhere)
	allArgs := append(args, outerArgs...)
	if err := r.db.Raw(countSQL, allArgs...).Scan(&filteredTotal).Error; err != nil {
		return nil, 0, summary, fmt.Errorf("gagal menghitung total data rekonsiliasi: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	finalSQL := fmt.Sprintf(`
SELECT * FROM (%s) AS t
%s
ORDER BY 
    CASE t.status_audit WHEN 'ANOMALI' THEN 1 WHEN 'PERINGATAN' THEN 2 ELSE 3 END,
    t.tgl DESC, t.unit ASC
LIMIT ? OFFSET ?
`, rawQuery, outerWhere)

	finalArgs := append(allArgs, limit, offset)
	if err := r.db.Raw(finalSQL, finalArgs...).Scan(&items).Error; err != nil {
		return nil, filteredTotal, summary, fmt.Errorf("gagal mengambil data rekonsiliasi: %w", err)
	}

	return items, filteredTotal, summary, nil
}

// GetDetailPoints retrieves 6 measurement points from dt_solar for crosscheck
func (r *solarRepository) GetDetailPoints(tgl string, unit string) ([]models.DtSolar, error) {
	var points []models.DtSolar
	err := r.db.Table("dt_solar").
		Where("tgl = ? AND unit = ?", tgl, unit).
		Order("jam ASC").
		Find(&points).Error
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil detail points dt_solar: %w", err)
	}
	return points, nil
}

// GetLaporanFilterOptions retrieves distinct areas, distinct units and min/max dates
func (r *solarRepository) GetLaporanFilterOptions() ([]string, []string, string, string, error) {
	var areas []string
	r.db.Table("dt_solar_unit").Where("area != '' AND area IS NOT NULL").Distinct("area").Order("area ASC").Pluck("area", &areas)

	var units []string
	r.db.Table("dt_solar_summary").Distinct("unit").Order("unit ASC").Pluck("unit", &units)

	// Combine with master units
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

	var minTgl, maxTgl string
	_ = r.db.Raw("SELECT COALESCE(DATE_FORMAT(MIN(tgl), '%Y-%m-%d'), ''), COALESCE(DATE_FORMAT(MAX(tgl), '%Y-%m-%d'), '') FROM dt_solar_summary").Row().Scan(&minTgl, &maxTgl)

	return areas, finalUnits, minTgl, maxTgl, nil
}
