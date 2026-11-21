package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"asm/models"
	"asm/repository"

	"github.com/xuri/excelize/v2"
)

type SolarService interface {
	GetPemakaianSolar(filter models.PemakaianSolarFilter) (map[string]interface{}, error)
	GetFilterOptions() (map[string]interface{}, error)

	// DtSolar Methods
	GetDtSolar(filter models.DtSolarFilter) (map[string]interface{}, error)
	GetDtSolarFilterOptions() (map[string]interface{}, error)
	ImportDtSolarFromRows(rows [][]string, defaultUnit string) (int, int, error)
	ImportDtSolarFromReader(r io.Reader, filename string, defaultUnit string) (int, int, error)
	ImportDtSolarFromSample() (int, int, error)
	TruncateDtSolar() error
	PostImportSync() error

	// Laporan Colokan Solar
	GetLaporanDetailHarian(filter models.LaporanDetailHarianFilter) (map[string]interface{}, error)
	GetLaporanRekapUnit(filter models.LaporanRekapUnitFilter) (map[string]interface{}, error)
	GetLaporanRekonsiliasi(filter models.LaporanRekonsiliasiFilter) (map[string]interface{}, error)
	GetLaporanMonitoringTangki(filter models.LaporanMonitoringTangkiFilter) (map[string]interface{}, error)
	GetDetailPoints(tgl string, unit string) ([]models.DtSolar, error)
	GetLaporanFilterOptions() (map[string]interface{}, error)

	// Dashboard Solar
	GetDashboardSolarUnits(filter models.DashboardSolarFilter) ([]models.DashboardSolarUnitCard, error)
	SubmitColokan(req models.InputColokanRequest) error
	SubmitIsiSolar(req models.InputIsiSolarRequest) error
}

type solarService struct {
	repo repository.SolarRepository
}

func NewSolarService(repo repository.SolarRepository) SolarService {
	return &solarService{repo: repo}
}

func (s *solarService) GetPemakaianSolar(filter models.PemakaianSolarFilter) (map[string]interface{}, error) {
	data, totalRows, err := s.repo.GetPemakaianSolar(filter)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetSummary(filter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"data":       data,
		"total_rows": totalRows,
		"page":       filter.Page,
		"limit":      filter.Limit,
		"summary":    summary,
	}, nil
}

func (s *solarService) GetFilterOptions() (map[string]interface{}, error) {
	units, shifts, minTgl, maxTgl, err := s.repo.GetFilterOptions()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"units":   units,
		"shifts":  shifts,
		"min_tgl": minTgl,
		"max_tgl": maxTgl,
	}, nil
}

// ==========================================
// DT_SOLAR IMPLEMENTATION
// ==========================================

func (s *solarService) GetDtSolar(filter models.DtSolarFilter) (map[string]interface{}, error) {
	data, totalRows, err := s.repo.GetDtSolar(filter)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetDtSolarSummary(filter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"data":       data,
		"total_rows": totalRows,
		"page":       filter.Page,
		"limit":      filter.Limit,
		"summary":    summary,
	}, nil
}

func (s *solarService) GetDtSolarFilterOptions() (map[string]interface{}, error) {
	units, jams, minTgl, maxTgl, err := s.repo.GetDtSolarFilterOptions()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"units":   units,
		"jams":    jams,
		"min_tgl": minTgl,
		"max_tgl": maxTgl,
	}, nil
}

func (s *solarService) TruncateDtSolar() error {
	return s.repo.TruncateDtSolar()
}

// ImportDtSolarFromRows processes rows of data, converting 1 row into 6 granular rows for dt_solar
func (s *solarService) ImportDtSolarFromRows(rows [][]string, defaultUnit string) (int, int, error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}

	var allRecords []models.DtSolar
	totalRowsProcessed := 0
	now := time.Now()

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		rawTgl := strings.TrimSpace(row[0])
		if rawTgl == "" || strings.EqualFold(rawTgl, "tanggal") || strings.EqualFold(rawTgl, "tgl") || strings.EqualFold(rawTgl, "date") {
			continue
		}

		parsedDate, err := parseDate(rawTgl)
		if err != nil {
			// Lewati baris jika kolom pertama bukan tanggal valid
			continue
		}

		// Unit dari Kolom Z (index 25)
		unit := defaultUnit
		if len(row) > 25 && strings.TrimSpace(row[25]) != "" {
			unit = strings.TrimSpace(row[25])
		}
		if unit == "" {
			unit = "SK330-8"
		}

		// Validasi apakah baris ini memiliki data (bukan baris kosong)
		hasData := false
		for c := 1; c < len(row) && c <= 25; c++ {
			v := strings.TrimSpace(row[c])
			if v != "" && v != "-" && v != "0" && v != "0.0" {
				hasData = true
				break
			}
		}

		// Tetap buat 6 data sesuai permintaan user
		totalRowsProcessed++

		// #data1: Jam 7 (Shift Pagi Mulai)
		// hm: kolom B (index 1), awal: kolom E (index 4), akhir: kolom E, isi: 0, satuan: liter
		allRecords = append(allRecords, models.DtSolar{
			Tgl:     &parsedDate,
			Unit:    unit,
			Jam:     7,
			Hm:      getFloat(row, 1),
			Awal:    getFloat(row, 4),
			Akhir:   getFloat(row, 4),
			Isi:     0,
			Satuan:  "liter",
			Created: &now,
		})

		// #data2: Jam 16 (Shift Pagi Selesai)
		// hm: kolom C (index 2), awal: kolom H (index 7), akhir: kolom H, isi: 0, satuan: liter
		allRecords = append(allRecords, models.DtSolar{
			Tgl:     &parsedDate,
			Unit:    unit,
			Jam:     16,
			Hm:      getFloat(row, 2),
			Awal:    getFloat(row, 7),
			Akhir:   getFloat(row, 7),
			Isi:     0,
			Satuan:  "liter",
			Created: &now,
		})

		// #data3: Jam 17 (Pengisian Siang)
		// hm: 0, awal: kolom H (index 7), akhir: kolom K (index 10), isi: kolom J (index 9), satuan: liter
		allRecords = append(allRecords, models.DtSolar{
			Tgl:     &parsedDate,
			Unit:    unit,
			Jam:     17,
			Hm:      0,
			Awal:    getFloat(row, 7),
			Akhir:   getFloat(row, 10),
			Isi:     getFloat(row, 9),
			Satuan:  "liter",
			Created: &now,
		})

		// #data4: Jam 18 (Shift Malam Mulai)
		// hm: kolom M (index 12), awal: kolom P (index 15), akhir: kolom P, isi: 0, satuan: liter
		allRecords = append(allRecords, models.DtSolar{
			Tgl:     &parsedDate,
			Unit:    unit,
			Jam:     18,
			Hm:      getFloat(row, 12),
			Awal:    getFloat(row, 15),
			Akhir:   getFloat(row, 15),
			Isi:     0,
			Satuan:  "liter",
			Created: &now,
		})

		// #data5: Jam 22 (Shift Malam Selesai)
		// hm: kolom N (index 13), awal: kolom S (index 18), akhir: kolom S, isi: 0, satuan: liter
		allRecords = append(allRecords, models.DtSolar{
			Tgl:     &parsedDate,
			Unit:    unit,
			Jam:     22,
			Hm:      getFloat(row, 13),
			Awal:    getFloat(row, 18),
			Akhir:   getFloat(row, 18),
			Isi:     0,
			Satuan:  "liter",
			Created: &now,
		})

		// #data6: Jam 23 (Pengisian Malam)
		// hm: 0, awal: kolom S (index 18), akhir: kolom V (index 21), isi: kolom U (index 20), satuan: liter
		allRecords = append(allRecords, models.DtSolar{
			Tgl:     &parsedDate,
			Unit:    unit,
			Jam:     23,
			Hm:      0,
			Awal:    getFloat(row, 18),
			Akhir:   getFloat(row, 21),
			Isi:     getFloat(row, 20),
			Satuan:  "liter",
			Created: &now,
		})
		_ = hasData
	}

	if len(allRecords) == 0 {
		return 0, 0, nil
	}

	countInserted, err := s.repo.UpsertBatchDtSolar(allRecords)
	if err != nil {
		return 0, 0, err
	}

	return totalRowsProcessed, countInserted, nil
}

// ImportDtSolarFromReader reads a file (.csv or .xlsx) and imports into dt_solar
func (s *solarService) ImportDtSolarFromReader(r io.Reader, filename string, defaultUnit string) (int, int, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	if ext == ".xlsx" || ext == ".xls" {
		f, err := excelize.OpenReader(r)
		if err != nil {
			return 0, 0, fmt.Errorf("gagal membaca file excel: %w", err)
		}
		defer f.Close()

		sheetList := f.GetSheetList()
		if len(sheetList) == 0 {
			return 0, 0, fmt.Errorf("file excel tidak memiliki sheet")
		}

		totalFileRows := 0
		totalDtRows := 0

		for _, sheetName := range sheetList {
			rows, err := f.GetRows(sheetName)
			if err != nil || len(rows) == 0 {
				continue
			}

			unitName := defaultUnit
			if unitName == "" {
				unitName = sheetName
			}

			fRows, dRows, err := s.ImportDtSolarFromRows(rows, unitName)
			if err != nil {
				return 0, 0, fmt.Errorf("gagal import sheet '%s': %w", sheetName, err)
			}
			totalFileRows += fRows
			totalDtRows += dRows
		}

		if totalDtRows > 0 {
			if syncErr := s.PostImportSync(); syncErr != nil {
				log.Printf("[SolarService] Warning PostImportSync error: %v", syncErr)
			}
		}

		return totalFileRows, totalDtRows, nil
	}

	// CSV Reader with automatic delimiter detection & BOM handling
	content, err := io.ReadAll(r)
	if err != nil {
		return 0, 0, fmt.Errorf("gagal membaca file CSV: %w", err)
	}

	// Strip UTF-8 BOM if present
	content = bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))

	comma := ','
	firstLine := ""
	if idx := bytes.IndexByte(content, '\n'); idx != -1 {
		firstLine = string(content[:idx])
	} else {
		firstLine = string(content)
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		comma = ';'
	}

	csvReader := csv.NewReader(bytes.NewReader(content))
	csvReader.Comma = comma
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := csvReader.ReadAll()
	if err != nil {
		return 0, 0, fmt.Errorf("gagal membaca format CSV: %w", err)
	}

	fRows, dRows, err := s.ImportDtSolarFromRows(records, defaultUnit)
	if err != nil {
		return 0, 0, err
	}
	if dRows > 0 {
		if syncErr := s.PostImportSync(); syncErr != nil {
			log.Printf("[SolarService] Warning PostImportSync error: %v", syncErr)
		}
	}
	return fRows, dRows, nil
}

// PostImportSync runs Query 1 and Query 2 after import data
func (s *solarService) PostImportSync() error {
	return s.repo.PostImportSync()
}

// ImportDtSolarFromSample imports from the stored sample file
func (s *solarService) ImportDtSolarFromSample() (int, int, error) {
	samplePath := "MyWork/sample_solar_import.csv"
	file, err := os.Open(samplePath)
	if err != nil {
		return 0, 0, fmt.Errorf("file sample tidak ditemukan: %w", err)
	}
	defer file.Close()

	return s.ImportDtSolarFromReader(file, "sample_solar_import.csv", "SK330-8")
}

// GetLaporanDetailHarian handles daily detail crosscheck logic
func (s *solarService) GetLaporanDetailHarian(filter models.LaporanDetailHarianFilter) (map[string]interface{}, error) {
	items, totalRows, summary, err := s.repo.GetLaporanDetailHarian(filter)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"data":       items,
		"total_rows": totalRows,
		"page":       filter.Page,
		"limit":      filter.Limit,
		"summary":    summary,
	}, nil
}

// GetLaporanRekapUnit handles unit operational evaluation recap logic
func (s *solarService) GetLaporanRekapUnit(filter models.LaporanRekapUnitFilter) (map[string]interface{}, error) {
	items, summary, err := s.repo.GetLaporanRekapUnit(filter)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"data":    items,
		"summary": summary,
	}, nil
}

// GetLaporanRekonsiliasi handles audit and finance reconciliation logic
func (s *solarService) GetLaporanRekonsiliasi(filter models.LaporanRekonsiliasiFilter) (map[string]interface{}, error) {
	items, totalRows, summary, err := s.repo.GetLaporanRekonsiliasi(filter)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"data":       items,
		"total_rows": totalRows,
		"page":       filter.Page,
		"limit":      filter.Limit,
		"summary":    summary,
	}, nil
}

// GetDetailPoints retrieves 6 hourly points for crosscheck
func (s *solarService) GetDetailPoints(tgl string, unit string) ([]models.DtSolar, error) {
	return s.repo.GetDetailPoints(tgl, unit)
}

// GetLaporanFilterOptions retrieves filter options
func (s *solarService) GetLaporanFilterOptions() (map[string]interface{}, error) {
	areas, units, minTgl, maxTgl, err := s.repo.GetLaporanFilterOptions()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"areas":   areas,
		"units":   units,
		"min_tgl": minTgl,
		"max_tgl": maxTgl,
	}, nil
}

// GetLaporanMonitoringTangki handles monitoring and tank days of supply logic
func (s *solarService) GetLaporanMonitoringTangki(filter models.LaporanMonitoringTangkiFilter) (map[string]interface{}, error) {
	tankSummaries, items, totalRows, projections, err := s.repo.GetLaporanMonitoringTangki(filter)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))

	return map[string]interface{}{
		"tank_summaries": tankSummaries,
		"items":          items,
		"projections":    projections,
		"pagination": map[string]interface{}{
			"current_page": page,
			"limit":        limit,
			"total_rows":   totalRows,
			"total_pages":  totalPages,
		},
	}, nil
}

// GetDashboardSolarUnits retrieves unit cards for Dashboard Solar
func (s *solarService) GetDashboardSolarUnits(filter models.DashboardSolarFilter) ([]models.DashboardSolarUnitCard, error) {
	return s.repo.GetDashboardSolarUnits(filter)
}

// SubmitColokan processes colokan input
func (s *solarService) SubmitColokan(req models.InputColokanRequest) error {
	return s.repo.SubmitColokan(req)
}

// SubmitIsiSolar processes fuel filling input
func (s *solarService) SubmitIsiSolar(req models.InputIsiSolarRequest) error {
	return s.repo.SubmitIsiSolar(req)
}
