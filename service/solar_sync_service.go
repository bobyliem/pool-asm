package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"asm/models"

	"gorm.io/gorm"
)

type TargetUnit struct {
	Unit string
	Gid  string
}

// Master daftar unit dan GID spreadsheet resmi
var TargetUnits = []TargetUnit{
	{Unit: "SK330-8", Gid: "891536450"},
	{Unit: "SK330-8 02", Gid: "1881427701"},
	{Unit: "SK330-8 03", Gid: "0"},
	{Unit: "SK330-8 04", Gid: "646613254"},
	{Unit: "SK330-8 05", Gid: "769000144"},
	{Unit: "SK330-8 06", Gid: "147785165"},
	{Unit: "SK200-10 05", Gid: "1214974238"},
	{Unit: "CAT 966G", Gid: "1685221017"},
	{Unit: "CAT 950G", Gid: "1673222375"},
	{Unit: "CAT 950G 02", Gid: "744285583"},
	{Unit: "KOB 03", Gid: "1883676271"},
	{Unit: "PC200-8", Gid: "490825643"},
	{Unit: "KOB 04", Gid: "85464892"},
}

// UnitSheetMapping maps unit names to Google Sheet GIDs
var UnitSheetMapping = map[string]string{
	"SK330-8":     "891536450",
	"SK330-8 02":  "1881427701",
	"SK330-8 03":  "0",
	"SK330-8 04":  "646613254",
	"SK330-8 05":  "769000144",
	"SK330-8 06":  "147785165",
	"SK200-10 05": "1214974238",
	"CAT 966G":    "1685221017",
	"CAT 950G":    "1673222375",
	"CAT 950G 02": "744285583",
	"KOB 03":      "1883676271",
	"PC200-8":     "490825643",
	"KOB 04":      "85464892",
}

const (
	defaultSpreadsheetID = "1T2XJiUlSZNq4x3jrT7w5w1mOUJo3EF5EYjcnvTi2ww4"
	defaultSK330Gid      = "891536450"
)

type SolarSyncService interface {
	TriggerSync(unitName string) (bool, string)
	GetStatus() models.SolarSyncSnapshot
}

type solarSyncService struct {
	db    *gorm.DB
	mutex sync.Mutex
	state models.SolarSyncSnapshot
}

func NewSolarSyncService(db *gorm.DB) SolarSyncService {
	// Auto migrate table pemakaian_solar if db is available
	if db != nil {
		if err := db.AutoMigrate(&models.PemakaianSolar{}); err != nil {
			log.Printf("[SolarSync] Warning auto migrate pemakaian_solar: %v", err)
		}
	}

	return &solarSyncService{
		db: db,
		state: models.SolarSyncSnapshot{
			Status:  "idle",
			Message: "Belum ada proses sync pemakaian solar yang dijalankan.",
			Logs:    make([]string, 0),
		},
	}
}

func (s *solarSyncService) TriggerSync(unitName string) (bool, string) {
	s.mutex.Lock()
	if s.state.IsRunning {
		s.mutex.Unlock()
		return false, "Proses Sync Solar masih sedang berjalan. Silakan tunggu hingga selesai."
	}

	now := time.Now()
	s.state.IsRunning = true
	s.state.StartedAt = &now
	s.state.FinishedAt = nil
	s.state.Status = "running"
	s.state.Unit = unitName
	s.state.Inserted = 0
	s.state.Updated = 0
	s.state.Total = 0

	label := unitName
	if label == "" || strings.EqualFold(label, "all") || strings.EqualFold(label, "Semua Unit") {
		label = "Semua Unit (13 Unit)"
	}

	s.state.Logs = []string{fmt.Sprintf("[%s] Memulai Sync Solar untuk %s...", now.Format("15:04:05"), label)}
	s.state.Message = fmt.Sprintf("Sedang menyinkronkan data pemakaian solar (%s)...", label)
	s.mutex.Unlock()

	go s.runSync(unitName)

	return true, fmt.Sprintf("Sync Solar untuk %s berhasil dimulai di background.", label)
}

func (s *solarSyncService) GetStatus() models.SolarSyncSnapshot {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logsCopy := make([]string, len(s.state.Logs))
	copy(logsCopy, s.state.Logs)

	snap := s.state
	snap.Logs = logsCopy
	return snap
}

func (s *solarSyncService) addLog(msg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	s.state.Logs = append(s.state.Logs, entry)
	if len(s.state.Logs) > 300 {
		s.state.Logs = s.state.Logs[len(s.state.Logs)-300:]
	}
}

func (s *solarSyncService) finishWithError(errMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	s.state.IsRunning = false
	s.state.FinishedAt = &now
	s.state.Status = "error"
	s.state.Message = errMsg
	s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] ERROR: %s", now.Format("15:04:05"), errMsg))
	log.Printf("[SolarSync] Error: %s", errMsg)
}

func (s *solarSyncService) finishWithSuccess(msg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	s.state.IsRunning = false
	s.state.FinishedAt = &now
	s.state.Status = "success"
	s.state.Message = msg
	s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] SELESAI: %s", now.Format("15:04:05"), msg))
	log.Printf("[SolarSync] Success: %s", msg)
}

func (s *solarSyncService) runSync(unitFilter string) {
	defer func() {
		if r := recover(); r != nil {
			s.finishWithError(fmt.Sprintf("Terjadi kesalahan sistem (panic): %v", r))
		}
	}()

	if s.db == nil {
		s.finishWithError("Koneksi database tidak tersedia.")
		return
	}

	var targets []TargetUnit

	cleanUnit := strings.TrimSpace(unitFilter)
	if cleanUnit == "" || strings.EqualFold(cleanUnit, "all") || strings.EqualFold(cleanUnit, "Semua Unit") || strings.Contains(strings.ToLower(cleanUnit), "semua") {
		// Sync SEMUA 13 Unit
		targets = TargetUnits
	} else if gid, ok := UnitSheetMapping[cleanUnit]; ok {
		targets = append(targets, TargetUnit{Unit: cleanUnit, Gid: gid})
	} else {
		// Fallback cari di TargetUnits case-insensitive
		found := false
		for _, tu := range TargetUnits {
			if strings.EqualFold(tu.Unit, cleanUnit) {
				targets = append(targets, tu)
				found = true
				break
			}
		}
		if !found {
			targets = append(targets, TargetUnit{Unit: cleanUnit, Gid: defaultSK330Gid})
		}
	}

	totalInserted := 0
	totalUpdated := 0
	totalRecords := 0

	client := &http.Client{Timeout: 45 * time.Second}

	for idx, target := range targets {
		s.mutex.Lock()
		s.state.Message = fmt.Sprintf("[%d/%d] Menyinkronkan %s...", idx+1, len(targets), target.Unit)
		s.mutex.Unlock()

		s.addLog(fmt.Sprintf("[%d/%d] Mengunduh data sheet unit %s (GID: %s)...", idx+1, len(targets), target.Unit, target.Gid))

		url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%s", defaultSpreadsheetID, target.Gid)
		resp, err := client.Get(url)
		if err != nil {
			s.addLog(fmt.Sprintf("PERINGATAN: Gagal mengakses spreadsheet untuk %s: %v", target.Unit, err))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			s.addLog(fmt.Sprintf("PERINGATAN: Google Sheets mengembalikan HTTP %d untuk %s", resp.StatusCode, target.Unit))
			continue
		}

		records, err := csv.NewReader(resp.Body).ReadAll()
		resp.Body.Close()
		if err != nil && err != io.EOF {
			s.addLog(fmt.Sprintf("PERINGATAN: Gagal membaca format CSV %s: %v", target.Unit, err))
			continue
		}

		// Validasi jumlah baris: mulai dari baris ke-12 (index 11)
		if len(records) < 12 {
			s.addLog(fmt.Sprintf("Sheet %s memiliki baris data kurang dari 12 (%d baris), dilewati.", target.Unit, len(records)))
			continue
		}

		// Cek nama unit di header baris ke-2 jika ada
		detectedUnitName := target.Unit
		if len(records) > 1 && len(records[1]) > 2 && strings.TrimSpace(records[1][2]) != "" {
			detectedUnitName = strings.TrimSpace(records[1][2])
		}

		sheetInserted := 0
		sheetUpdated := 0

		// Mulai dari baris ke 12 (index 11)
		for rowIdx := 11; rowIdx < len(records); rowIdx++ {
			row := records[rowIdx]
			if len(row) == 0 {
				continue
			}

			// Tanggal di kolom A (index 0)
			rawTanggal := strings.TrimSpace(row[0])
			if rawTanggal == "" {
				continue
			}

			parsedDate, err := parseDate(rawTanggal)
			if err != nil {
				// Bukan tanggal valid, lewati
				continue
			}

			// Value Unit dari kolom Z (index 25), fallback ke target.Unit
			unitVal := target.Unit
			if len(row) > 25 && strings.TrimSpace(row[25]) != "" {
				unitVal = strings.TrimSpace(row[25])
			} else if detectedUnitName != "" {
				unitVal = detectedUnitName
			}

			// Helper: Cek apakah shift memiliki data operasional (bukan sel kosong di sheet)
			hasShiftData := func(startCol, endCol int) bool {
				for c := startCol; c <= endCol && c < len(row); c++ {
					val := strings.TrimSpace(row[c])
					if val != "" && val != "-" {
						return true
					}
				}
				return false
			}

			pagiAdaData := hasShiftData(1, 11)
			malamAdaData := hasShiftData(12, 22)

			// Lewati baris template kosong (misal placeholder tanggal bulan depan yang belum ada isinya)
			if !pagiAdaData && !malamAdaData {
				continue
			}

			// ==========================================
			// 1. Shift Pagi: Kolom A s/d L (index 0..11)
			// ==========================================
			if pagiAdaData {
				pagiAwalHm := getFloat(row, 1)
				pagiAkhirHm := getFloat(row, 2)
				pagiTotalJamHm := getFloat(row, 3)
				pagiColokanAwal := getFloat(row, 4)  // Colokan Pagi
				pagiTotalPakaiCm := getFloat(row, 5) // Total Pakai CM
				pagiPakaiLiter := getFloat(row, 6)   // Pakai Siang Liter
				pagiColokanAkhir := getFloat(row, 7) // Colokan Sore
				pagiTotalIsiCm := getFloat(row, 8)   // Total Isi CM
				pagiIsiLiter := getFloat(row, 9)     // Isi Siang Liter
				pagiAkhirSetelahIsi := getFloat(row, 10)
				pagiSisaTangki := getFloat(row, 11)

				var pagiRasio float64
				if pagiTotalJamHm > 0 {
					pagiRasio = pagiPakaiLiter / pagiTotalJamHm
				}

				pagiModel := models.PemakaianSolar{
					Unit:              unitVal,
					Tanggal:           parsedDate,
					Shift:             "Pagi",
					AwalHm:            pagiAwalHm,
					AkhirHm:           pagiAkhirHm,
					TotalJamHm:        pagiTotalJamHm,
					ColokanAwalCm:     pagiColokanAwal,
					TotalPakaiCm:      pagiTotalPakaiCm,
					PakaiLiter:        pagiPakaiLiter,
					ColokanAkhirCm:    pagiColokanAkhir,
					TotalIsiCm:        pagiTotalIsiCm,
					IsiLiter:          pagiIsiLiter,
					AkhirSetelahIsiCm: pagiAkhirSetelahIsi,
					SisaTangkiLiter:   pagiSisaTangki,
					RasioLiterPerHm:   pagiRasio,
				}

				resPagi, err := s.upsertRecord(pagiModel)
				if err == nil {
					if resPagi == "inserted" {
						sheetInserted++
					} else if resPagi == "updated" {
						sheetUpdated++
					}
				}
			}

			// ==========================================
			// 2. Shift Malam: Kolom M s/d W (index 12..22)
			// ==========================================
			if malamAdaData {
				malamAwalHm := getFloat(row, 12)
				malamAkhirHm := getFloat(row, 13)
				malamTotalJamHm := getFloat(row, 14)
				malamColokanAwal := getFloat(row, 15)  // Colokan Malam
				malamTotalPakaiCm := getFloat(row, 16) // Total Pakai CM
				malamPakaiLiter := getFloat(row, 17)   // Pakai Malam Liter
				malamColokanAkhir := getFloat(row, 18) // Colokan Pagi
				malamTotalIsiCm := getFloat(row, 19)   // Total Isi CM
				malamIsiLiter := getFloat(row, 20)     // Isi Malam Liter
				malamAkhirSetelahIsi := getFloat(row, 21)
				malamSisaTangki := getFloat(row, 22)

				var malamRasio float64
				if malamTotalJamHm > 0 {
					malamRasio = malamPakaiLiter / malamTotalJamHm
				}

				malamModel := models.PemakaianSolar{
					Unit:              unitVal,
					Tanggal:           parsedDate,
					Shift:             "Malam",
					AwalHm:            malamAwalHm,
					AkhirHm:           malamAkhirHm,
					TotalJamHm:        malamTotalJamHm,
					ColokanAwalCm:     malamColokanAwal,
					TotalPakaiCm:      malamTotalPakaiCm,
					PakaiLiter:        malamPakaiLiter,
					ColokanAkhirCm:    malamColokanAkhir,
					TotalIsiCm:        malamTotalIsiCm,
					IsiLiter:          malamIsiLiter,
					AkhirSetelahIsiCm: malamAkhirSetelahIsi,
					SisaTangkiLiter:   malamSisaTangki,
					RasioLiterPerHm:   malamRasio,
				}

				resMalam, err := s.upsertRecord(malamModel)
				if err == nil {
					if resMalam == "inserted" {
						sheetInserted++
					} else if resMalam == "updated" {
						sheetUpdated++
					}
				}
			}
		}

		s.addLog(fmt.Sprintf("[%d/%d] Selesai %s: %d baru, %d diperbarui.", idx+1, len(targets), target.Unit, sheetInserted, sheetUpdated))
		totalInserted += sheetInserted
		totalUpdated += sheetUpdated
		totalRecords += (sheetInserted + sheetUpdated)

		// Update running stats
		s.mutex.Lock()
		s.state.Inserted = totalInserted
		s.state.Updated = totalUpdated
		s.state.Total = totalRecords
		s.mutex.Unlock()
	}

	// Bersihkan record template kosong (phantom rows di masa depan) yang seluruh nilainya 0
	s.db.Exec(`DELETE FROM pemakaian_solar 
		WHERE awal_hm = 0 AND akhir_hm = 0 AND total_jam_hm = 0 
		  AND colokan_awal_cm = 0 AND total_pakai_cm = 0 AND pakai_liter = 0 
		  AND total_isi_cm = 0 AND isi_liter = 0 AND sisa_tangki_liter = 0`)

	s.finishWithSuccess(fmt.Sprintf("Sinkronisasi %d unit selesai: %d data baru ditambahkan, %d data diperbarui.", len(targets), totalInserted, totalUpdated))
}

func (s *solarSyncService) upsertRecord(record models.PemakaianSolar) (string, error) {
	now := time.Now()
	var existing models.PemakaianSolar
	err := s.db.Where("unit = ? AND tanggal = ? AND shift = ?", record.Unit, record.Tanggal.Format("2006-01-02"), record.Shift).First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			record.CreatedAt = &now
			record.UpdatedAt = &now
			if errCreate := s.db.Create(&record).Error; errCreate != nil {
				return "", errCreate
			}
			return "inserted", nil
		}
		return "", err
	}

	// Update record yang sudah ada
	existing.AwalHm = record.AwalHm
	existing.AkhirHm = record.AkhirHm
	existing.TotalJamHm = record.TotalJamHm
	existing.ColokanAwalCm = record.ColokanAwalCm
	existing.TotalPakaiCm = record.TotalPakaiCm
	existing.PakaiLiter = record.PakaiLiter
	existing.ColokanAkhirCm = record.ColokanAkhirCm
	existing.TotalIsiCm = record.TotalIsiCm
	existing.IsiLiter = record.IsiLiter
	existing.AkhirSetelahIsiCm = record.AkhirSetelahIsiCm
	existing.SisaTangkiLiter = record.SisaTangkiLiter
	existing.RasioLiterPerHm = record.RasioLiterPerHm
	existing.UpdatedAt = &now

	if errUpdate := s.db.Save(&existing).Error; errUpdate != nil {
		return "", errUpdate
	}
	return "updated", nil
}

func getFloat(row []string, idx int) float64 {
	if idx >= len(row) {
		return 0
	}
	val := strings.TrimSpace(row[idx])
	if val == "" || val == "-" {
		return 0
	}
	// Ganti koma ke titik jika format angka Indonesia (misal 14743,3)
	val = strings.ReplaceAll(val, ",", ".")
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}
	return f
}

// parseDate handles formats such as "1 Sep 2026", "01 Sep 2026", "2026-09-01", etc.
func parseDate(val string) (time.Time, error) {
	s := strings.TrimSpace(val)

	// Normalisasi singkatan bulan bahasa Indonesia ke Inggris
	replacer := strings.NewReplacer(
		"Agu", "Aug",
		"Okt", "Oct",
		"Des", "Dec",
		"Mei", "May",
	)
	sNorm := replacer.Replace(s)

	formats := []string{
		"2 Jan 2006",
		"02 Jan 2006",
		"2 Jan 06",
		"02 Jan 06",
		"2006-01-02",
		"02-01-2006",
		"02/01/2006",
		"2/1/2006",
		"2-1-2006",
	}

	for _, fmtStr := range formats {
		if t, err := time.ParseInLocation(fmtStr, sNorm, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("format tanggal tidak dikenali: %s", val)
}
