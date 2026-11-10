package service

import (
	"asm/models"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"asm/repository"

	"github.com/xuri/excelize/v2"
)

type ClosingFinanceService interface {
	GetData(filter models.ClosingFinanceFilter) ([]models.ExtClosing, int64, models.ClosingFinanceSummary, error)
	GetFilterOptions() (map[string]interface{}, error)
	ProcessExcelUpload(file io.Reader) (int, error)
	Truncate() error
}

type closingFinanceService struct {
	repo repository.ClosingFinanceRepository
}

func NewClosingFinanceService(repo repository.ClosingFinanceRepository) ClosingFinanceService {
	return &closingFinanceService{repo: repo}
}

func (s *closingFinanceService) GetData(filter models.ClosingFinanceFilter) ([]models.ExtClosing, int64, models.ClosingFinanceSummary, error) {
	return s.repo.GetAll(filter)
}

func (s *closingFinanceService) GetFilterOptions() (map[string]interface{}, error) {
	cust, cat, tp, err := s.repo.GetFilterOptions()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"customers":  cust,
		"categories": cat,
		"types":      tp,
	}, nil
}

func (s *closingFinanceService) Truncate() error {
	return s.repo.Truncate()
}

func parseDateTime(val string) *time.Time {
	val = strings.TrimSpace(val)
	if val == "" || val == "-" || val == "NULL" {
		return nil
	}

	layouts := []string{
		"02-Jan-2006 15:04",
		"02-Jan-2006 15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, val); err == nil {
			return &t
		}
	}

	return nil
}

func parseFloat(val string) float64 {
	val = strings.TrimSpace(val)
	if val == "" || val == "-" {
		return 0
	}
	// Hapus koma pemisah ribuan dan karakter non angka (kecuali minus dan titik)
	cleaned := strings.ReplaceAll(val, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "Rp", "")
	cleaned = strings.ReplaceAll(cleaned, "rp", "")

	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return f
}

func (s *closingFinanceService) ProcessExcelUpload(file io.Reader) (int, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, fmt.Errorf("gagal membaca file Excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, fmt.Errorf("file Excel tidak memiliki sheet")
	}

	// Baca sheet pertama
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return 0, fmt.Errorf("gagal membaca baris sheet: %w", err)
	}

	if len(rows) < 2 {
		return 0, fmt.Errorf("file Excel kosong atau hanya berisi header")
	}

	// Deteksi mapping kolom dari header (row 0)
	headerMap := make(map[string]int)
	for colIdx, colName := range rows[0] {
		norm := strings.ToLower(strings.TrimSpace(colName))
		norm = strings.ReplaceAll(norm, " ", "")
		norm = strings.ReplaceAll(norm, "_", "")
		norm = strings.ReplaceAll(norm, "/", "")
		headerMap[norm] = colIdx
	}

	getColIdx := func(keys ...string) int {
		for _, k := range keys {
			norm := strings.ToLower(strings.TrimSpace(k))
			norm = strings.ReplaceAll(norm, " ", "")
			norm = strings.ReplaceAll(norm, "_", "")
			norm = strings.ReplaceAll(norm, "/", "")
			if idx, ok := headerMap[norm]; ok {
				return idx
			}
		}
		return -1
	}

	idxCust := getColIdx("customer", "cust")
	idxSO := getColIdx("serviceorder", "spk", "service_order")
	idxCat := getColIdx("category", "kategori")
	idxReff := getColIdx("reference", "reff")
	idxNopol := getColIdx("platenumber", "nopol", "plate_number")
	idxInfo := getColIdx("information", "info")
	idxReleased := getColIdx("releasedat", "released", "tglmasuk")
	idxCompleted := getColIdx("completedat", "completed", "exitdate")
	idxType := getColIdx("productservice", "tipe", "type")
	idxCode := getColIdx("code", "sku")
	idxName := getColIdx("name", "prd", "product")
	idxNotes := getColIdx("notes", "note")
	idxQty := getColIdx("qty", "quantity")
	idxAvg := getColIdx("averagecost", "average")
	idxPrice := getColIdx("unitprice", "harga", "price")
	idxTotal := getColIdx("total")

	var items []models.ExtClosing
	now := time.Now()

	for rIdx := 1; rIdx < len(rows); rIdx++ {
		row := rows[rIdx]
		if len(row) == 0 {
			continue
		}

		getVal := func(idx int) string {
			if idx >= 0 && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}

		soVal := getVal(idxSO)
		nopolVal := getVal(idxNopol)
		custVal := getVal(idxCust)

		// Lewati baris jika spk, nopol, dan customer semuanya kosong
		if soVal == "" && nopolVal == "" && custVal == "" {
			continue
		}

		relAt := parseDateTime(getVal(idxReleased))
		compAt := parseDateTime(getVal(idxCompleted))

		statueVal := "Completed"
		if compAt == nil {
			statueVal = "Open"
		}

		item := models.ExtClosing{
			Cust:         custVal,
			ServiceOrder: soVal,
			Category:     getVal(idxCat),
			Reff:         getVal(idxReff),
			Nopol:        nopolVal,
			Information:  getVal(idxInfo),
			Released:     relAt,
			Completed:    compAt,
			Tipe:         getVal(idxType),
			Sku:          getVal(idxCode),
			Prd:          getVal(idxName),
			Notes:        getVal(idxNotes),
			Qty:          parseFloat(getVal(idxQty)),
			Average:      parseFloat(getVal(idxAvg)),
			Harga:        parseFloat(getVal(idxPrice)),
			Total:        parseFloat(getVal(idxTotal)),
			Statue:       statueVal,
			Created:      &now,
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		return 0, fmt.Errorf("tidak ada data yang valid ditemukan di file Excel")
	}

	// Truncate and insert
	if err := s.repo.TruncateAndInsert(items); err != nil {
		return 0, fmt.Errorf("gagal menyimpan data ke database: %w", err)
	}

	return len(items), nil
}
