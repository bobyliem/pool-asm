package repository

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"asm/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SolarRepository interface {
	GetPemakaianSolar(filter models.PemakaianSolarFilter) ([]models.DtSolarHisItem, int64, error)
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
	GetLaporanMonitoringTangki(filter models.LaporanMonitoringTangkiFilter) ([]models.TangkiStatusSummary, []models.LaporanMonitoringTangkiItem, int64, []models.TangkiProjectionItem, []string, error)
	GetDetailPoints(tgl string, unit string) ([]models.DtSolar, error)
	GetLaporanFilterOptions() ([]string, []string, string, string, error)

	// Dashboard Solar
	GetDashboardSolarUnits(filter models.DashboardSolarFilter) ([]models.DashboardSolarUnitCard, error)
	SubmitColokan(req models.InputColokanRequest) error
	SubmitIsiSolar(req models.InputIsiSolarRequest) error
}

type solarRepository struct {
	db *gorm.DB
}

func NewSolarRepository(db *gorm.DB) SolarRepository {
	return &solarRepository{db: db}
}

func (r *solarRepository) buildPemakaianWhere(filter models.PemakaianSolarFilter) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}

	if filter.TglAwal != "" {
		where += " AND h.tgl >= ?"
		args = append(args, filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		where += " AND h.tgl <= ?"
		args = append(args, filter.TglAkhir)
	}

	cleanUnit := strings.TrimSpace(filter.Unit)
	if cleanUnit != "" && !strings.EqualFold(cleanUnit, "all") && !strings.EqualFold(cleanUnit, "Semua Unit") {
		where += " AND h.unit = ?"
		args = append(args, cleanUnit)
	}

	cleanNote := strings.TrimSpace(filter.Note)
	if cleanNote != "" && !strings.EqualFold(cleanNote, "all") && !strings.EqualFold(cleanNote, "Semua Note") {
		where += " AND h.note = ?"
		args = append(args, cleanNote)
	}

	return where, args
}

func (r *solarRepository) GetPemakaianSolar(filter models.PemakaianSolarFilter) ([]models.DtSolarHisItem, int64, error) {
	var results []models.DtSolarHisItem
	var totalRows int64

	whereClause, args := r.buildPemakaianWhere(filter)

	// Count total rows
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM dt_solar_his h %s`, whereClause)
	if err := r.db.Raw(countSQL, args...).Scan(&totalRows).Error; err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung total data: %w", err)
	}

	// Paginated data
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 1000 {
		limit = 50
	}
	offset := (page - 1) * limit

	querySQL := fmt.Sprintf(`
SELECT
    h.id,
    DATE_FORMAT(h.tgl, '%%Y-%%m-%%d') AS tanggal,
    h.jam,
    h.unit,
    h.hm AS hm_awal,
    h.hm_end AS hm_akhir,
    ROUND(GREATEST(h.hm_end - h.hm, 0), 2) AS total_hm,
    ROUND(h.awal, 2) AS saldo_awal,
    ROUND(h.isi, 2) AS isi,
    ROUND(h.isi_input, 2) AS isi_input,
    ROUND(h.akhir, 2) AS saldo_akhir,
    COALESCE(h.note, '') AS note,
    DATE_FORMAT(h.created, '%%Y-%%m-%%d %%H:%%i:%%s') AS created
FROM dt_solar_his h
%s
ORDER BY h.tgl DESC, h.jam DESC, h.id DESC
LIMIT ? OFFSET ?
`, whereClause)

	queryArgs := append(args, limit, offset)
	if err := r.db.Raw(querySQL, queryArgs...).Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("gagal mengambil data pemakaian solar: %w", err)
	}

	return results, totalRows, nil
}

func (r *solarRepository) GetSummary(filter models.PemakaianSolarFilter) (models.SolarSummary, error) {
	var summary models.SolarSummary

	whereClause, args := r.buildPemakaianWhere(filter)

	summarySQL := fmt.Sprintf(`
SELECT
    COUNT(*) AS total_rows,
    ROUND(COALESCE(SUM(CASE WHEN h.isi < 0 THEN -h.isi ELSE 0 END), 0), 2) AS total_pakai_liter,
    ROUND(COALESCE(SUM(CASE WHEN h.isi > 0 THEN h.isi ELSE 0 END), 0), 2) AS total_isi_liter,
    ROUND(COALESCE(SUM(CASE WHEN h.hm_end > h.hm THEN h.hm_end - h.hm ELSE 0 END), 0), 2) AS total_jam_hm
FROM dt_solar_his h
%s
`, whereClause)

	type AggResult struct {
		TotalRows       int64   `gorm:"column:total_rows"`
		TotalPakaiLiter float64 `gorm:"column:total_pakai_liter"`
		TotalIsiLiter   float64 `gorm:"column:total_isi_liter"`
		TotalJamHm      float64 `gorm:"column:total_jam_hm"`
	}

	var agg AggResult
	if err := r.db.Raw(summarySQL, args...).Scan(&agg).Error; err != nil {
		return summary, fmt.Errorf("gagal menghitung summary: %w", err)
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
	// Unique units from dt_solar_his
	var units []string
	r.db.Raw("SELECT DISTINCT unit FROM dt_solar_his ORDER BY unit ASC").Scan(&units)

	// Unique notes from dt_solar_his (excluding empty notes)
	var notes []string
	r.db.Raw("SELECT DISTINCT note FROM dt_solar_his WHERE note != '' ORDER BY note ASC").Scan(&notes)

	// Min and Max tanggal
	var minTglStr, maxTglStr string
	r.db.Raw("SELECT COALESCE(MIN(DATE_FORMAT(tgl, '%Y-%m-%d')), '') FROM dt_solar_his").Scan(&minTglStr)
	r.db.Raw("SELECT COALESCE(MAX(DATE_FORMAT(tgl, '%Y-%m-%d')), '') FROM dt_solar_his").Scan(&maxTglStr)

	return units, notes, minTglStr, maxTglStr, nil
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

	baseWhere := "WHERE 1=1 AND COALESCE(u.tangki, 'No') = 'No'"
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

	baseWhere := "WHERE 1=1 AND COALESCE(u.tangki, 'No') = 'No'"
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

	baseWhere := "WHERE 1=1 AND COALESCE(u.tangki, 'No') = 'No'"
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
	r.db.Table("dt_solar_unit").Where("tangki = ?", "No").Distinct("unit").Order("unit ASC").Pluck("unit", &units)

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

// GetLaporanMonitoringTangki retrieves tank status overview, daily ledger, and forward projections
func (r *solarRepository) GetLaporanMonitoringTangki(filter models.LaporanMonitoringTangkiFilter) ([]models.TangkiStatusSummary, []models.LaporanMonitoringTangkiItem, int64, []models.TangkiProjectionItem, []string, error) {
	// 1. Ambil list tangki (u.tangki = 'Yes')
	tankQuery := r.db.Table("dt_solar_unit").Where("tangki = ?", "Yes")
	if filter.Area != "" {
		tankQuery = tankQuery.Where("area = ?", filter.Area)
	}
	if filter.Tangki != "" {
		tankQuery = tankQuery.Where("unit = ?", filter.Tangki)
	}

	type TankRow struct {
		Unit string
		Area string
	}
	var tankRows []TankRow
	if err := tankQuery.Select("unit, area").Order("area ASC, unit ASC").Scan(&tankRows).Error; err != nil {
		return nil, nil, 0, nil, nil, err
	}

	now := time.Now()
	var tankSummaries []models.TangkiStatusSummary
	var projections []models.TangkiProjectionItem

	// 2. Evaluasi status & ketahanan stok tiap tangki
	for _, tr := range tankRows {
		summary := models.TangkiStatusSummary{
			Unit: tr.Unit,
			Area: tr.Area,
		}

		// Ambil saldo terakhir dari dt_solar_summary
		var latestStock float64
		var lastDate string
		err := r.db.Raw(`
			SELECT saldoakhir, COALESCE(DATE_FORMAT(tgl, '%Y-%m-%d'), '')
			FROM dt_solar_summary
			WHERE unit = ?
			ORDER BY tgl DESC
			LIMIT 1
		`, tr.Unit).Row().Scan(&latestStock, &lastDate)
		if err == nil {
			summary.CurrentStock = latestStock
			summary.LastDate = lastDate
		}

		// Jumlah unit yang mengambil suplai dari tangki ini (induk = tank.id)
		var unitCount int
		_ = r.db.Raw("SELECT COUNT(*) FROM dt_solar_unit WHERE induk = (SELECT id FROM dt_solar_unit WHERE unit = ? LIMIT 1)", tr.Unit).Row().Scan(&unitCount)
		summary.UnitCount = unitCount

		// Pengisian/Refill terakhir tangki dari supplier
		var lastRefillDate string
		var lastRefillQty float64
		err = r.db.Raw(`
			SELECT COALESCE(DATE_FORMAT(tgl, '%Y-%m-%d'), ''), isi
			FROM dt_solar_tangki
			WHERE unit = ? AND isi > 0
			ORDER BY tgl DESC
			LIMIT 1
		`, tr.Unit).Row().Scan(&lastRefillDate, &lastRefillQty)
		if err != nil || lastRefillQty == 0 {
			// fallback ke dt_solar_summary
			_ = r.db.Raw(`
				SELECT COALESCE(DATE_FORMAT(tgl, '%Y-%m-%d'), ''), isi
				FROM dt_solar_summary
				WHERE unit = ? AND isi > 0
				ORDER BY tgl DESC
				LIMIT 1
			`, tr.Unit).Row().Scan(&lastRefillDate, &lastRefillQty)
		}
		summary.LastRefillDate = lastRefillDate
		summary.LastRefillQty = lastRefillQty

		// Hitung burn rate rata-rata (pemakaian harian tangki ke unit)
		// Mengambil 30 hari data terakhir yang aktif (pakai > 0)
		var totalPakai float64
		var activeDays int
		_ = r.db.Raw(`
			SELECT COALESCE(SUM(pakai), 0), COUNT(CASE WHEN pakai > 0 THEN 1 END)
			FROM dt_solar_summary
			WHERE unit = ? AND tgl >= DATE_SUB(COALESCE((SELECT MAX(tgl) FROM dt_solar_summary WHERE unit = ?), CURDATE()), INTERVAL 30 DAY)
		`, tr.Unit, tr.Unit).Row().Scan(&totalPakai, &activeDays)

		if activeDays > 0 && totalPakai > 0 {
			summary.ActiveDays = activeDays
			summary.AvgDailyBurn = math.Round((totalPakai/float64(activeDays))*10) / 10
		} else {
			summary.AvgDailyBurn = 0
		}

		// Hitung Ketahanan Stok (Days of Supply)
		if summary.AvgDailyBurn > 0 && summary.CurrentStock > 0 {
			summary.DaysOfSupply = math.Round((summary.CurrentStock/summary.AvgDailyBurn)*10) / 10
			daysToAdd := int(summary.DaysOfSupply)
			summary.RunOutDate = now.AddDate(0, 0, daysToAdd).Format("2006-01-02")
		} else if summary.CurrentStock <= 0 {
			summary.DaysOfSupply = 0
			summary.RunOutDate = now.Format("2006-01-02")
		} else {
			summary.DaysOfSupply = 999
			summary.RunOutDate = "-"
		}

		// Status alert
		if summary.DaysOfSupply <= 2 {
			summary.Status = "KRITIS"
			summary.StatusColor = "rose"
		} else if summary.DaysOfSupply <= 5 {
			summary.Status = "WASPADA"
			summary.StatusColor = "amber"
		} else {
			summary.Status = "AMAN"
			summary.StatusColor = "emerald"
		}

		tankSummaries = append(tankSummaries, summary)

		// 3. Proyeksi 14 hari ke depan untuk tangki ini
		curSaldo := summary.CurrentStock
		burnRate := summary.AvgDailyBurn
		for d := 1; d <= 14; d++ {
			pDate := now.AddDate(0, 0, d).Format("2006-01-02")
			nextSaldo := curSaldo - burnRate
			if nextSaldo < 0 {
				nextSaldo = 0
			}

			sisaHari := 0.0
			if burnRate > 0 {
				sisaHari = math.Round((nextSaldo/burnRate)*10) / 10
			}

			pStatus := "AMAN"
			pColor := "emerald"
			if nextSaldo <= 0 {
				pStatus = "HABIS"
				pColor = "rose"
			} else if sisaHari <= 2 {
				pStatus = "KRITIS"
				pColor = "rose"
			} else if sisaHari <= 5 {
				pStatus = "WASPADA"
				pColor = "amber"
			}

			projections = append(projections, models.TangkiProjectionItem{
				HariKe:       d,
				Tanggal:      pDate,
				UnitTangki:   tr.Unit,
				Area:         tr.Area,
				SaldoAwal:    math.Round(curSaldo*10) / 10,
				EstimasiBurn: math.Round(burnRate*10) / 10,
				SaldoAkhir:   math.Round(nextSaldo*10) / 10,
				SisaHari:     sisaHari,
				Status:       pStatus,
				StatusColor:  pColor,
			})

			curSaldo = nextSaldo
		}
	}

	// 4. Query Riwayat Mutasi Harian Tangki (Ledger)
	cleanTangki := strings.TrimSpace(filter.Tangki)
	if cleanTangki == "" && strings.EqualFold(filter.Area, "Cicangkal") {
		cleanTangki = "TANGKI-CCK"
	} else if cleanTangki == "" && strings.EqualFold(filter.Area, "Pion") {
		cleanTangki = "TANGKI-PION"
	}

	// Pas Tab All diklik: tidak usah munculkan data row
	if cleanTangki == "" {
		return tankSummaries, []models.LaporanMonitoringTangkiItem{}, 0, projections, []string{}, nil
	}

	// Cari data unit tangki dan unit-unit yang induknya adalah tangki ini
	var tankID int
	var tankArea string
	err := r.db.Raw("SELECT id, area FROM dt_solar_unit WHERE unit = ? LIMIT 1", cleanTangki).Row().Scan(&tankID, &tankArea)
	if err != nil {
		return tankSummaries, nil, 0, projections, nil, fmt.Errorf("tangki %s tidak ditemukan: %w", cleanTangki, err)
	}

	var relatedUnits []string
	err = r.db.Raw(`
		SELECT unit 
		FROM dt_solar_unit 
		WHERE induk = ? 
		ORDER BY tangki ASC, unit ASC
	`, tankID).Scan(&relatedUnits).Error
	if err != nil {
		return tankSummaries, nil, 0, projections, nil, fmt.Errorf("gagal mengambil unit terkait: %w", err)
	}

	// Query data mutasi BBM tangki
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "s.unit = ?")
	args = append(args, cleanTangki)

	if filter.TglAwal != "" {
		conditions = append(conditions, "s.tgl >= ?")
		args = append(args, filter.TglAwal)
	}
	if filter.TglAkhir != "" {
		conditions = append(conditions, "s.tgl <= ?")
		args = append(args, filter.TglAkhir)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM dt_solar_summary s
		JOIN dt_solar_unit u ON s.unit = u.unit
		%s
	`, whereClause)

	var totalRows int64
	if err := r.db.Raw(countQuery, args...).Row().Scan(&totalRows); err != nil {
		return tankSummaries, nil, 0, projections, relatedUnits, err
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

	dataQuery := fmt.Sprintf(`
		SELECT s.id, 
		       COALESCE(DATE_FORMAT(s.tgl, '%%Y-%%m-%%d'), '') as tgl,
		       s.unit, 
		       COALESCE(u.area, '-') as area,
		       ROUND(COALESCE(s.saldoawal, 0), 1) as saldo_awal,
		       ROUND(COALESCE(s.isi, 0), 1) as isi,
		       ROUND(COALESCE(s.pakai, 0), 1) as pakai,
		       ROUND(COALESCE(s.saldoakhir, 0), 1) as saldo_akhir
		FROM dt_solar_summary s
		JOIN dt_solar_unit u ON s.unit = u.unit
		%s
		ORDER BY s.tgl DESC
		LIMIT %d OFFSET %d
	`, whereClause, limit, offset)

	var items []models.LaporanMonitoringTangkiItem
	if err := r.db.Raw(dataQuery, args...).Scan(&items).Error; err != nil {
		return tankSummaries, nil, totalRows, projections, relatedUnits, err
	}

	// Query isi dan pakai unit terkait pada tanggal-tanggal tersebut
	if len(items) > 0 && len(relatedUnits) > 0 {
		var tglList []string
		for _, it := range items {
			if it.Tgl != "" {
				tglList = append(tglList, it.Tgl)
			}
		}

		type UnitSummaryRow struct {
			Tgl   string  `gorm:"column:tgl"`
			Unit  string  `gorm:"column:unit"`
			Isi   float64 `gorm:"column:isi"`
			Pakai float64 `gorm:"column:pakai"`
		}
		var unitRows []UnitSummaryRow

		_ = r.db.Raw(`
			SELECT COALESCE(DATE_FORMAT(tgl, '%Y-%m-%d'), '') as tgl,
			       unit,
			       ROUND(COALESCE(isi, 0), 1) as isi,
			       ROUND(COALESCE(pakai, 0), 1) as pakai
			FROM dt_solar_summary
			WHERE unit IN (?) AND tgl IN (?)
		`, relatedUnits, tglList).Scan(&unitRows).Error

		// Buat lookup map: dateUnitMap[tgl][unit] = UnitIsiPakai
		dateUnitMap := make(map[string]map[string]models.UnitIsiPakai)
		for _, ur := range unitRows {
			if _, ok := dateUnitMap[ur.Tgl]; !ok {
				dateUnitMap[ur.Tgl] = make(map[string]models.UnitIsiPakai)
			}
			dateUnitMap[ur.Tgl][ur.Unit] = models.UnitIsiPakai{
				Isi:   ur.Isi,
				Pakai: ur.Pakai,
			}
		}

		for i := range items {
			var totIsi float64
			var totPakai float64
			detail := make(map[string]models.UnitIsiPakai)

			for _, ru := range relatedUnits {
				var val models.UnitIsiPakai
				if dMap, ok := dateUnitMap[items[i].Tgl]; ok {
					val = dMap[ru]
				}
				detail[ru] = val
				totIsi += val.Isi
				totPakai += val.Pakai
			}

			items[i].TotalIsiUnit = math.Round(totIsi*10) / 10
			items[i].TotalPakaiUnit = math.Round(totPakai*10) / 10
			items[i].UnitsDetail = detail
		}
	}

	return tankSummaries, items, totalRows, projections, relatedUnits, nil
}

// GetDashboardSolarUnits retrieves unit cards for Dashboard Solar
func (r *solarRepository) GetDashboardSolarUnits(filter models.DashboardSolarFilter) ([]models.DashboardSolarUnitCard, error) {
	whereClause := "WHERE 1=1"
	var queryArgs []interface{}

	cleanArea := strings.TrimSpace(filter.Area)
	if cleanArea != "" && !strings.EqualFold(cleanArea, "all") && !strings.EqualFold(cleanArea, "Semua Area") {
		if strings.EqualFold(cleanArea, "Cicangkal") {
			whereClause += " AND (u.area = 'Cicangkal' OR parent.unit = 'TANGKI-CCK')"
		} else {
			whereClause += " AND u.area = ?"
			queryArgs = append(queryArgs, cleanArea)
		}
	}

	cleanTgl := strings.TrimSpace(filter.Tanggal)

	var items []models.DashboardSolarUnitCard

	if cleanTgl != "" {
		selectArgs := []interface{}{cleanTgl, cleanTgl, cleanTgl, cleanTgl, cleanTgl, cleanTgl}
		allArgs := append(selectArgs, queryArgs...)

		query := fmt.Sprintf(`
		SELECT 
			u.id, u.unit, u.area, u.tangki, u.induk,
			COALESCE(parent.unit, CASE WHEN u.induk = 0 THEN 'Supplier Luar' ELSE '-' END) as induk_name,
			ROUND(COALESCE(s.hm_akhir, u.hm), 3) as hm, 
			ROUND(COALESCE(s.saldoakhir, u.sisasolar), 2) as sisa_solar, 
			ROUND(u.konsumsi, 2) as konsumsi,
			ROUND(COALESCE(s.pakai, 0), 2) as hari_pakai,
			ROUND(COALESCE(s.isi, 0), 2) as hari_isi,
			COALESCE(DATE_FORMAT(u.updated, '%%Y-%%m-%%d %%H:%%i:%%s'), '-') as updated,
			COALESCE((
				SELECT CASE WHEN SUM(sm.hm_akhir - sm.hm_awal) > 0 THEN ROUND(SUM(sm.pakai) / SUM(sm.hm_akhir - sm.hm_awal), 2) ELSE 0 END
				FROM dt_solar_summary sm
				WHERE sm.unit = u.unit AND sm.tgl <= ? AND sm.tgl >= DATE_SUB(?, INTERVAL 30 DAY)
			), 0) as avg_liter_per_hm,
			COALESCE((
				SELECT DATE_FORMAT(h.tgl, '%%Y-%%m-%%d')
				FROM dt_solar_his h
				WHERE h.unit = u.unit AND h.tgl <= ?
				ORDER BY h.tgl DESC, h.jam DESC, h.id DESC
				LIMIT 1
			), '') as last_tgl,
			COALESCE((
				SELECT h.jam
				FROM dt_solar_his h
				WHERE h.unit = u.unit AND h.tgl <= ?
				ORDER BY h.tgl DESC, h.jam DESC, h.id DESC
				LIMIT 1
			), 0) as last_jam,
			COALESCE((
				SELECT h.note
				FROM dt_solar_his h
				WHERE h.unit = u.unit AND h.tgl <= ?
				ORDER BY h.tgl DESC, h.jam DESC, h.id DESC
				LIMIT 1
			), '') as last_note
		FROM dt_solar_unit u
		LEFT JOIN dt_solar_unit parent ON u.induk = parent.id
		LEFT JOIN dt_solar_summary s ON u.unit = s.unit AND s.tgl = ?
		%s
		ORDER BY u.tangki ASC, u.induk ASC, u.unit ASC
	`, whereClause)

		if err := r.db.Raw(query, allArgs...).Scan(&items).Error; err != nil {
			return nil, fmt.Errorf("gagal mengambil data dashboard solar: %w", err)
		}

		for i := range items {
			if items[i].Tangki == "Yes" {
				items[i].Status = "TANGKI INDUK"

				// Hitung rata-rata pengeluaran solar harian (AVG L/Hari) & Ketahanan Stok (Hari)
				var totalPakai float64
				var activeDays int
				_ = r.db.Raw(`
					SELECT COALESCE(SUM(pakai), 0), COUNT(CASE WHEN pakai > 0 THEN 1 END)
					FROM dt_solar_summary
					WHERE unit = ? AND tgl <= ? AND tgl >= DATE_SUB(?, INTERVAL 30 DAY)
				`, items[i].Unit, cleanTgl, cleanTgl).Row().Scan(&totalPakai, &activeDays)

				if activeDays > 0 && totalPakai > 0 {
					items[i].AvgDailyBurn = math.Round((totalPakai/float64(activeDays))*10) / 10
				} else {
					items[i].AvgDailyBurn = 0
				}

				if items[i].AvgDailyBurn > 0 && items[i].SisaSolar > 0 {
					items[i].DaysOfSupply = math.Round((items[i].SisaSolar/items[i].AvgDailyBurn)*10) / 10
				} else if items[i].SisaSolar <= 0 {
					items[i].DaysOfSupply = 0
				} else {
					items[i].DaysOfSupply = 999
				}
			} else if items[i].HariPakai > 0 || items[i].HariIsi > 0 {
				items[i].Status = "BEROPERASI"
			} else if items[i].AvgLiterPerHm > 0 || items[i].SisaSolar > 0 {
				items[i].Status = "BEROPERASI"
			} else {
				items[i].Status = "STANDBY"
			}
		}

		// Urutkan berdasarkan status asc (standby di urutan bawah), dan nama unit
		sort.Slice(items, func(i, j int) bool {
			rank := func(item models.DashboardSolarUnitCard) int {
				if item.Tangki == "Yes" {
					return 1
				}
				if item.Status == "STANDBY" {
					return 3
				}
				return 2 // BEROPERASI
			}

			rankI := rank(items[i])
			rankJ := rank(items[j])
			if rankI != rankJ {
				return rankI < rankJ
			}
			return items[i].Unit < items[j].Unit
		})
	} else {
		query := fmt.Sprintf(`
		SELECT 
			u.id, u.unit, u.area, u.tangki, u.induk,
			COALESCE(parent.unit, CASE WHEN u.induk = 0 THEN 'Supplier Luar' ELSE '-' END) as induk_name,
			ROUND(u.hm, 3) as hm, 
			ROUND(u.sisasolar, 2) as sisa_solar, 
			ROUND(u.konsumsi, 2) as konsumsi,
			0 as hari_pakai,
			0 as hari_isi,
			COALESCE(DATE_FORMAT(u.updated, '%%Y-%%m-%%d %%H:%%i:%%s'), '-') as updated,
			COALESCE((
				SELECT CASE WHEN SUM(s.hm_akhir - s.hm_awal) > 0 THEN ROUND(SUM(s.pakai) / SUM(s.hm_akhir - s.hm_awal), 2) ELSE 0 END
				FROM dt_solar_summary s
				WHERE s.unit = u.unit AND s.tgl >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)
			), 0) as avg_liter_per_hm,
			COALESCE((
				SELECT DATE_FORMAT(h.tgl, '%%Y-%%m-%%d')
				FROM dt_solar_his h
				WHERE h.unit = u.unit
				ORDER BY h.tgl DESC, h.jam DESC, h.id DESC
				LIMIT 1
			), '') as last_tgl,
			COALESCE((
				SELECT h.jam
				FROM dt_solar_his h
				WHERE h.unit = u.unit
				ORDER BY h.tgl DESC, h.jam DESC, h.id DESC
				LIMIT 1
			), 0) as last_jam,
			COALESCE((
				SELECT h.note
				FROM dt_solar_his h
				WHERE h.unit = u.unit
				ORDER BY h.tgl DESC, h.jam DESC, h.id DESC
				LIMIT 1
			), '') as last_note
		FROM dt_solar_unit u
		LEFT JOIN dt_solar_unit parent ON u.induk = parent.id
		%s
		ORDER BY u.tangki ASC, u.induk ASC, u.unit ASC
	`, whereClause)

		if err := r.db.Raw(query, queryArgs...).Scan(&items).Error; err != nil {
			return nil, fmt.Errorf("gagal mengambil data dashboard solar: %w", err)
		}

		for i := range items {
			if items[i].Tangki == "Yes" {
				items[i].Status = "TANGKI INDUK"

				// Hitung rata-rata pengeluaran solar harian (AVG L/Hari) & Ketahanan Stok (Hari)
				var totalPakai float64
				var activeDays int
				_ = r.db.Raw(`
					SELECT COALESCE(SUM(pakai), 0), COUNT(CASE WHEN pakai > 0 THEN 1 END)
					FROM dt_solar_summary
					WHERE unit = ? AND tgl >= DATE_SUB(COALESCE((SELECT MAX(tgl) FROM dt_solar_summary WHERE unit = ?), CURDATE()), INTERVAL 30 DAY)
				`, items[i].Unit, items[i].Unit).Row().Scan(&totalPakai, &activeDays)

				if activeDays > 0 && totalPakai > 0 {
					items[i].AvgDailyBurn = math.Round((totalPakai/float64(activeDays))*10) / 10
				} else {
					items[i].AvgDailyBurn = 0
				}

				if items[i].AvgDailyBurn > 0 && items[i].SisaSolar > 0 {
					items[i].DaysOfSupply = math.Round((items[i].SisaSolar/items[i].AvgDailyBurn)*10) / 10
				} else if items[i].SisaSolar <= 0 {
					items[i].DaysOfSupply = 0
				} else {
					items[i].DaysOfSupply = 999
				}
			} else if items[i].AvgLiterPerHm > 0 || items[i].SisaSolar > 0 {
				items[i].Status = "BEROPERASI"
			} else {
				items[i].Status = "STANDBY"
			}
		}

		// Urutkan berdasarkan status asc (standby di urutan bawah), dan nama unit
		sort.Slice(items, func(i, j int) bool {
			rank := func(item models.DashboardSolarUnitCard) int {
				if item.Tangki == "Yes" {
					return 1
				}
				if item.Status == "STANDBY" {
					return 3
				}
				return 2 // BEROPERASI
			}

			rankI := rank(items[i])
			rankJ := rank(items[j])
			if rankI != rankJ {
				return rankI < rankJ
			}
			return items[i].Unit < items[j].Unit
		})
	}

	return items, nil
}

// SubmitColokan executes CALL sp_solar for colokan input
func (r *solarRepository) SubmitColokan(req models.InputColokanRequest) error {
	// 1. Cek unit di dt_solar_unit
	type UnitCheck struct {
		ID        int
		Hm        float64
		Tangki    string
		Konsumsi  float64
		Sisasolar float64
	}
	var u UnitCheck
	err := r.db.Raw("SELECT id, hm, tangki, konsumsi, sisasolar FROM dt_solar_unit WHERE unit = ? LIMIT 1", req.Unit).Scan(&u).Error
	if err != nil || u.ID == 0 {
		return fmt.Errorf("unit '%s' tidak ditemukan di master dt_solar_unit", req.Unit)
	}

	if u.Tangki == "Yes" {
		return fmt.Errorf("unit '%s' adalah tangki induk, transaksi colokan hanya untuk unit alat berat", req.Unit)
	}

	// 2. Validasi Colokan Solar tidak boleh lebih besar dari saldo sebelumnya
	sisaCm := 0.0
	if u.Konsumsi > 0 {
		sisaCm = u.Sisasolar / u.Konsumsi
	}
	if req.CmSesudah > (sisaCm + 0.05) {
		return fmt.Errorf("Colokan Solar lebih besar dari saldo sebelumnya!")
	}

	// 3. Validasi HM
	if req.Hm < u.Hm {
		return fmt.Errorf("HM input (%.3f) tidak boleh lebih kecil dari HM unit saat ini (%.3f)", req.Hm, u.Hm)
	}
	if u.Hm > 0 && req.Hm > (2*u.Hm) {
		return fmt.Errorf("HM tidak normal, yang diinput lebih besar dari HM sebelumnya!")
	}

	// 4. Validasi Jam terhadap pengisian data terakhir di dt_solar_his
	var lastJam int
	_ = r.db.Raw("SELECT COALESCE(MAX(jam), 0) FROM dt_solar_his WHERE unit = ? AND tgl = ?", req.Unit, req.Tanggal).Row().Scan(&lastJam)
	if lastJam > 0 && req.Jam <= lastJam {
		return fmt.Errorf("pengisian data terakhir pada tanggal %s adalah jam %02d:00. Jam input (%02d:00) harus lebih besar", req.Tanggal, lastJam, req.Jam)
	}

	// 5. Eksekusi Stored Procedure
	// CALL sp_solar(tanggal, jam, unit, hm, 0, 0, cm_sesudah)
	callSQL := "CALL sp_solar(?, ?, ?, ?, 0, 0, ?)"
	if err := r.db.Exec(callSQL, req.Tanggal, req.Jam, req.Unit, req.Hm, req.CmSesudah).Error; err != nil {
		return fmt.Errorf("gagal mengeksekusi sp_solar: %w", err)
	}

	return nil
}

// SubmitIsiSolar executes CALL sp_solar for fuel filling input
func (r *solarRepository) SubmitIsiSolar(req models.InputIsiSolarRequest) error {
	// 1. Cek unit di dt_solar_unit
	type UnitCheck struct {
		ID       int
		Hm       float64
		Tangki   string
		Konsumsi float64
	}
	var u UnitCheck
	err := r.db.Raw("SELECT id, hm, tangki, konsumsi FROM dt_solar_unit WHERE unit = ? LIMIT 1", req.Unit).Scan(&u).Error
	if err != nil || u.ID == 0 {
		return fmt.Errorf("unit '%s' tidak ditemukan di master dt_solar_unit", req.Unit)
	}

	// Khusus Unit Tangki: TANGKI-CCK dan TANGKI-PION
	if u.Tangki == "Yes" {
		if req.IsiLiter <= 0 {
			return fmt.Errorf("jumlah pengisian (Liter) harus lebih besar dari 0")
		}

		// Validasi Jam terhadap pengisian data terakhir di dt_solar_his
		var lastJam int
		_ = r.db.Raw("SELECT COALESCE(MAX(jam), 0) FROM dt_solar_his WHERE unit = ? AND tgl = ?", req.Unit, req.Tanggal).Row().Scan(&lastJam)
		if lastJam > 0 && req.Jam <= lastJam {
			return fmt.Errorf("pengisian data terakhir pada tanggal %s adalah jam %02d:00. Jam input (%02d:00) harus lebih besar", req.Tanggal, lastJam, req.Jam)
		}

		// CALL sp_tangki_sedot(tanggal, jam, 14, isi_liter, isi_liter, 15, 'FillUnit');
		// 14 = default value; 15 = dt_solar_unit.id (TANGKI-PION=15, TANGKI-CCK=14)
		callSQL := "CALL sp_tangki_sedot(?, ?, 14, ?, ?, ?, 'FillUnit')"
		if err := r.db.Exec(callSQL, req.Tanggal, req.Jam, req.IsiLiter, req.IsiLiter, u.ID).Error; err != nil {
			return fmt.Errorf("gagal mengeksekusi sp_tangki_sedot: %w", err)
		}
		return nil
	}

	// 2. Validasi HM
	if req.Hm < u.Hm {
		return fmt.Errorf("HM input (%.3f) tidak boleh lebih kecil dari HM unit saat ini (%.3f)", req.Hm, u.Hm)
	}
	if u.Hm > 0 && req.Hm > (2*u.Hm) {
		return fmt.Errorf("HM tidak normal, yang diinput lebih besar dari HM sebelumnya!")
	}

	// 3. Validasi Jam terhadap pengisian data terakhir di dt_solar_his
	var lastJam int
	_ = r.db.Raw("SELECT COALESCE(MAX(jam), 0) FROM dt_solar_his WHERE unit = ? AND tgl = ?", req.Unit, req.Tanggal).Row().Scan(&lastJam)
	if lastJam > 0 && req.Jam <= lastJam {
		return fmt.Errorf("pengisian data terakhir pada tanggal %s adalah jam %02d:00. Jam input (%02d:00) harus lebih besar", req.Tanggal, lastJam, req.Jam)
	}

	// 4. Eksekusi Stored Procedure
	cmSebelum := req.CmAwal
	if cmSebelum == 0 && req.CmSebelum != 0 {
		cmSebelum = req.CmSebelum
	}
	cmSesudah := req.CmAkhir
	if cmSesudah == 0 && req.CmSesudah != 0 {
		cmSesudah = req.CmSesudah
	}

	// CALL sp_solar(tanggal, jam, unit, hm, cm_sebelum, isi_liter, cm_sesudah)
	callSQL := "CALL sp_solar(?, ?, ?, ?, ?, ?, ?)"
	if err := r.db.Exec(callSQL, req.Tanggal, req.Jam, req.Unit, req.Hm, cmSebelum, req.IsiLiter, cmSesudah).Error; err != nil {
		return fmt.Errorf("gagal mengeksekusi sp_solar: %w", err)
	}

	return nil
}
