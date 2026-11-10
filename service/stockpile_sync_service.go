package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"asm/models"

	"gorm.io/gorm"
)

// =====================================================
// Stockpile Sync Service
// Pulls "Last Movement" data from Metabase dashboard
// and upserts into api_stockpile table.
// =====================================================

const (
	metabaseDashboardUUID = "a3bf4989-89d9-45ac-a6e2-6a9d906165fa"
	metabaseDashcardID    = "45"
	metabaseCardID        = "43"
	metabaseBaseURL       = "https://dashboard.sampurna-group.com"

	// Dashboard parameter IDs
	paramStartDateID = "737441b3"
	paramEndDateID   = "20a4546"
)

// StockpileSyncService defines the interface for stockpile synchronization
type StockpileSyncService interface {
	TriggerSync(customStartDate, customEndDate string) (bool, string)
	GetStatus() StockpileSyncSnapshot
}

// StockpileSyncSnapshot is the public status snapshot
type StockpileSyncSnapshot struct {
	IsRunning  bool       `json:"is_running"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	StartDate  string     `json:"start_date"`
	EndDate    string     `json:"end_date"`
	Inserted   int        `json:"inserted"`
	Updated    int        `json:"updated"`
	Skipped    int        `json:"skipped"`
	Total      int        `json:"total"`
	Logs       []string   `json:"logs"`
}

type stockpileSyncState struct {
	sync.Mutex
	IsRunning  bool
	StartedAt  *time.Time
	FinishedAt *time.Time
	Status     string
	Message    string
	StartDate  string
	EndDate    string
	Inserted   int
	Updated    int
	Skipped    int
	Total      int
	Logs       []string
}

type stockpileSyncService struct {
	db    *gorm.DB
	state stockpileSyncState
}

func NewStockpileSyncService(db *gorm.DB) StockpileSyncService {
	return &stockpileSyncService{
		db: db,
		state: stockpileSyncState{
			Status:  "idle",
			Message: "Belum ada proses sync stockpile yang dijalankan.",
			Logs:    make([]string, 0),
		},
	}
}

func (s *stockpileSyncService) TriggerSync(customStartDate, customEndDate string) (bool, string) {
	s.state.Lock()
	if s.state.IsRunning {
		s.state.Unlock()
		return false, "Proses sync stockpile masih berjalan. Silakan tunggu hingga selesai."
	}

	now := time.Now()
	startDate := customStartDate
	endDate := customEndDate

	// Default: selalu ambil data dari 7 hari lalu sampai hari ini jika parameter tidak diisi
	if startDate == "" {
		startDate = now.AddDate(0, 0, -7).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}

	s.state.IsRunning = true
	s.state.StartedAt = &now
	s.state.FinishedAt = nil
	s.state.Status = "running"
	s.state.StartDate = startDate
	s.state.EndDate = endDate
	s.state.Message = fmt.Sprintf("Memulai sync stockpile periode %s s/d %s...", startDate, endDate)
	s.state.Inserted = 0
	s.state.Updated = 0
	s.state.Skipped = 0
	s.state.Total = 0
	s.state.Logs = []string{fmt.Sprintf("[%s] Memulai sync stockpile periode %s s/d %s...", now.Format("15:04:05"), startDate, endDate)}
	s.state.Unlock()

	go s.runSync(startDate, endDate)

	return true, fmt.Sprintf("Sync stockpile periode %s s/d %s berhasil dijalankan di background.", startDate, endDate)
}

func (s *stockpileSyncService) GetStatus() StockpileSyncSnapshot {
	s.state.Lock()
	defer s.state.Unlock()

	logsCopy := make([]string, len(s.state.Logs))
	copy(logsCopy, s.state.Logs)

	return StockpileSyncSnapshot{
		IsRunning:  s.state.IsRunning,
		StartedAt:  s.state.StartedAt,
		FinishedAt: s.state.FinishedAt,
		Status:     s.state.Status,
		Message:    s.state.Message,
		StartDate:  s.state.StartDate,
		EndDate:    s.state.EndDate,
		Inserted:   s.state.Inserted,
		Updated:    s.state.Updated,
		Skipped:    s.state.Skipped,
		Total:      s.state.Total,
		Logs:       logsCopy,
	}
}

// runSync fetches data and upserts into DB
func (s *stockpileSyncService) runSync(startDate, endDate string) {
	defer func() {
		if r := recover(); r != nil {
			s.finishWithError(fmt.Sprintf("Panic pada proses sync: %v", r))
		}
	}()

	s.addLog(fmt.Sprintf("Menghubungi Metabase untuk data %s s/d %s...", startDate, endDate))

	// Fetch CSV from Metabase
	rows, err := s.fetchFromMetabase(startDate, endDate)
	if err != nil {
		s.finishWithError(fmt.Sprintf("Gagal fetch dari Metabase: %v", err))
		return
	}

	s.addLog(fmt.Sprintf("Berhasil mengambil %d rows dari Metabase.", len(rows)))

	if len(rows) == 0 {
		s.finishSuccess("Tidak ada data movement pada rentang tanggal tersebut.")
		return
	}

	if len(rows) >= 500 {
		s.addLog("PERINGATAN: Jumlah data mencapai 500 rows (Metabase query limit). Disarankan memperkecil rentang tanggal.")
	}

	// Upsert each row
	inserted, updated, skipped := 0, 0, 0
	for _, row := range rows {
		action, err := s.upsertRow(row)
		if err != nil {
			s.addLog(fmt.Sprintf("Error upsert tiket %s (%s): %v", row.NoTiket, row.Dk, err))
			skipped++
			continue
		}
		switch action {
		case "inserted":
			inserted++
		case "updated":
			updated++
		case "skipped":
			skipped++
		}
	}

	s.state.Lock()
	s.state.Inserted = inserted
	s.state.Updated = updated
	s.state.Skipped = skipped
	s.state.Total = len(rows)
	s.state.Unlock()

	summary := fmt.Sprintf("Sync selesai! Total: %d, Baru: %d, Diperbarui: %d, Sama (lewati): %d",
		len(rows), inserted, updated, skipped)
	s.addLog(summary)
	s.finishSuccess(summary)
}

// fetchFromMetabase calls the Metabase public API and returns parsed rows
func (s *stockpileSyncService) fetchFromMetabase(startDate, endDate string) ([]models.ApiStockpile, error) {
	url := fmt.Sprintf("%s/api/public/dashboard/%s/dashcard/%s/card/%s/csv",
		metabaseBaseURL, metabaseDashboardUUID, metabaseDashcardID, metabaseCardID)

	// Build request body
	paramsJSON := fmt.Sprintf(`[{"id":"%s","type":"date/single","target":["variable",["template-tag","startDate"]],"value":"%s"},{"id":"%s","type":"date/single","target":["variable",["template-tag","endDate"]],"value":"%s"}]`,
		paramStartDateID, startDate, paramEndDateID, endDate)

	body := fmt.Sprintf(`{"parameters":%s}`, paramsJSON)

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim request ke server Metabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse CSV response
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca format CSV dari response: %w", err)
	}

	if len(records) < 2 {
		return nil, nil // Hanya header atau kosong
	}

	// Map header columns
	header := records[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		colIndex[strings.TrimSpace(h)] = i
	}

	var result []models.ApiStockpile
	for _, rec := range records[1:] {
		row := models.ApiStockpile{
			NoTiket: getCol(rec, colIndex, "Ticket No"),
			Reff:    getCol(rec, colIndex, "Reference No"),
			Nopol:   getCol(rec, colIndex, "Plate No"),
			Dk:      getCol(rec, colIndex, "Type"),
			Prd:     getCol(rec, colIndex, "Product"),
			Src:     getCol(rec, colIndex, "Source"),
			Dest:    getCol(rec, colIndex, "Destination"),
			Wsrc:    getColFloat(rec, colIndex, "Weight"),
			Wdest:   getColFloat(rec, colIndex, "Weight Plant"),
		}

		// Skip row jika no_tiket kosong
		if row.NoTiket == "" {
			continue
		}

		// Parse date (Metabase format: "2026-10-31T17:39:12")
		dateStr := getCol(rec, colIndex, "Date")
		if dateStr != "" {
			dateStr = strings.Replace(dateStr, "T", " ", 1)
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr, time.Local); err == nil {
				row.Tgl = &t
			}
		}

		result = append(result, row)
	}

	return result, nil
}

// upsertRow checks if no_tiket exists, compares data, inserts or updates accordingly
func (s *stockpileSyncService) upsertRow(row models.ApiStockpile) (string, error) {
	var existing models.ApiStockpile
	query := s.db.Where("no_tiket = ?", row.NoTiket)
	if row.Dk != "" {
		query = query.Where("dk = ?", row.Dk)
	}

	result := query.First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// Data baru — lakukan INSERT
		now := time.Now()
		row.Created = &now
		if err := s.db.Create(&row).Error; err != nil {
			return "", fmt.Errorf("insert failed: %w", err)
		}
		return "inserted", nil
	}

	if result.Error != nil {
		return "", fmt.Errorf("query error: %w", result.Error)
	}

	// Data tiket sudah ada — bandingkan setiap kolom
	if s.hasChanges(existing, row) {
		now := time.Now()
		updates := map[string]interface{}{
			"reff":    row.Reff,
			"nopol":   row.Nopol,
			"dk":      row.Dk,
			"prd":     row.Prd,
			"src":     row.Src,
			"dest":    row.Dest,
			"wsrc":    row.Wsrc,
			"wdest":   row.Wdest,
			"tgl":     row.Tgl,
			"updated": &now, // Catat timestamp update saat terjadi perubahan data
		}
		if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
			return "", fmt.Errorf("update failed: %w", err)
		}
		return "updated", nil
	}

	// Data sama persis, tidak perlu update
	return "skipped", nil
}

// hasChanges compares existing DB record with new data from Metabase
func (s *stockpileSyncService) hasChanges(existing, incoming models.ApiStockpile) bool {
	if existing.Reff != incoming.Reff {
		return true
	}
	if existing.Nopol != incoming.Nopol {
		return true
	}
	if existing.Dk != incoming.Dk {
		return true
	}
	if existing.Prd != incoming.Prd {
		return true
	}
	if existing.Src != incoming.Src {
		return true
	}
	if existing.Dest != incoming.Dest {
		return true
	}
	if math.Abs(existing.Wsrc-incoming.Wsrc) > 0.0001 {
		return true
	}
	if math.Abs(existing.Wdest-incoming.Wdest) > 0.0001 {
		return true
	}

	// Bandingkan tanggal/waktu transaksi
	if existing.Tgl == nil && incoming.Tgl != nil {
		return true
	}
	if existing.Tgl != nil && incoming.Tgl == nil {
		return true
	}
	if existing.Tgl != nil && incoming.Tgl != nil && existing.Tgl.Unix() != incoming.Tgl.Unix() {
		return true
	}

	return false
}

// Helper functions
func getCol(rec []string, colIndex map[string]int, colName string) string {
	if idx, ok := colIndex[colName]; ok && idx < len(rec) {
		return strings.TrimSpace(rec[idx])
	}
	return ""
}

func getColFloat(rec []string, colIndex map[string]int, colName string) float64 {
	s := getCol(rec, colIndex, colName)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("[StockpileSync] Warning: Gagal parse angka '%s' pada kolom '%s': %v", s, colName, err)
		return 0
	}
	return v
}

func (s *stockpileSyncService) addLog(msg string) {
	s.state.Lock()
	defer s.state.Unlock()
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	s.state.Logs = append(s.state.Logs, entry)
	if len(s.state.Logs) > 100 {
		s.state.Logs = s.state.Logs[len(s.state.Logs)-100:]
	}
	s.state.Message = msg
	log.Println("[StockpileSync]", msg)
}

func (s *stockpileSyncService) finishWithError(msg string) {
	s.state.Lock()
	defer s.state.Unlock()
	now := time.Now()
	s.state.IsRunning = false
	s.state.FinishedAt = &now
	s.state.Status = "error"
	s.state.Message = msg
	s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] ERROR: %s", now.Format("15:04:05"), msg))
	log.Println("[StockpileSync] ERROR:", msg)
}

func (s *stockpileSyncService) finishSuccess(msg string) {
	s.state.Lock()
	defer s.state.Unlock()
	now := time.Now()
	s.state.IsRunning = false
	s.state.FinishedAt = &now
	s.state.Status = "success"
	s.state.Message = msg
	s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] SELESAI: %s", now.Format("15:04:05"), msg))
	log.Println("[StockpileSync] SUCCESS:", msg)
}
