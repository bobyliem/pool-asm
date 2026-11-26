package service

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type SyncState struct {
	sync.Mutex
	IsRunning    bool       `json:"is_running"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Status       string     `json:"status"` // "idle", "running", "success", "error"
	Message      string     `json:"message"`
	CurrentTable string     `json:"current_table"`
	Logs         []string   `json:"logs"`
}

type CsvSyncService interface {
	TriggerSync(table string) (bool, string)
	GetStatus() SyncStateSnapshot
}

type SyncStateSnapshot struct {
	IsRunning    bool       `json:"is_running"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Status       string     `json:"status"`
	Message      string     `json:"message"`
	CurrentTable string     `json:"current_table"`
	Logs         []string   `json:"logs"`
}

type colMapEntry struct {
	SourceKey string // Header CSV atau huruf kolom (A, B, F...)
	TargetCol string // Nama kolom di tabel MySQL
}

type importItemConfig struct {
	Table      string
	URL        string
	Truncate   bool
	SkipRows   int
	IsByLetter bool
	Columns    []colMapEntry
}

type importResult struct {
	Table    string
	Success  bool
	Inserted int
	Skipped  int
	Errors   []string
}

type csvSyncService struct {
	state SyncState
	db    *gorm.DB
}

func NewCsvSyncService(db *gorm.DB) CsvSyncService {
	return &csvSyncService{
		db: db,
		state: SyncState{
			Status:  "idle",
			Message: "Belum ada proses sinkronisasi yang dijalankan.",
			Logs:    make([]string, 0),
		},
	}
}

var (
	reTZOffset = regexp.MustCompile(`(\.\d+)?([+-]\d{2}:\d{2}|Z)$`)
	wibLoc     = loadWIBLocation()
)

func loadWIBLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

func getImportConfigs() []importItemConfig {
	soColumns := []colMapEntry{
		{"Visitor Id", "visitor_id"},
		{"id", "so_id"},
		{"Service Order", "service_order"},
		{"Category", "category"},
		{"Reference", "reff"},
		{"Customer", "customer"},
		{"Customer Code", "customer_code"},
		{"Site", "site"},
		{"Information", "information"},
		{"Plate Number", "plate_number"},
		{"Body Number", "body_number"},
		{"Vehicle Type", "vehicle_type"},
		{"Vehicle Category", "vehicle_category"},
		{"Status", "statue"},
		{"Released at", "released_at"},
		{"Completed at", "completed_at"},
		{"Type", "tipe"},
		{"Fulfillment", "fulfillment"},
		{"SKU", "sku"},
		{"Product", "product"},
		{"Qty", "qty"},
		{"Unit Price", "unit_price"},
		{"Total Price", "total_price"},
		{"Invoice", "invoice"},
		{"Service Order Id", "service_order_id"},
		{"Last Notes", "last_notes"},
	}

	return []importItemConfig{
		// 1. imp_visitor
		{
			Table:    "imp_visitor",
			URL:      "https://dashboard.sampurna-group.com/api/public/card/72fdec9c-3a83-43cc-880e-d92e17fc7457/query/csv",
			Truncate: true,
			Columns: []colMapEntry{
				{"Plate Number", "nopol"},
				{"Body Number", "nobak"},
				{"Purpose", "purpose"},
				{"Information", "information"},
				{"Customer", "customer"},
				{"Customer Code", "cust_code"},
				{"Formation", "formation"},
				{"Planner", "planner"},
				{"Driver", "driver"},
				{"Current Driver", "curr_driver"},
				{"Type", "tipe"},
				{"Category", "category"},
				{"Entry Date", "entrydate"},
				{"Entry By", "entryby"},
				{"Exit Date", "exitdate"},
				{"Exit By", "exitby"},
				{"id", "visitor_id"},
				{"Site", "sitename"},
			},
		},
		// 2. imp_service_order
		{
			Table:    "imp_service_order",
			URL:      "https://dashboard.sampurna-group.com/api/public/card/ba72218a-6281-478a-b84a-368dbca6872d/query/csv",
			Truncate: true,
			Columns:  soColumns,
		},
		// 3. imp_service_order_bam
		{
			Table:    "imp_service_order_bam",
			URL:      "https://dashboard.sampurna-group.com/api/public/card/2d1fe94b-04a4-48ac-985a-a1611eeaf417/query/csv",
			Truncate: true,
			Columns:  soColumns,
		},
		// 4. imp_stb
		{
			Table:    "imp_stb",
			URL:      "https://dashboard.sampurna-group.com/api/public/card/2df6816c-32ec-4942-8a1a-0dab2660557b/query/csv",
			Truncate: true,
			Columns: []colMapEntry{
				{"No. STB", "no_stb"},
				{"No. PO", "no_po"},
				{"Terms of Payment", "terms_of_payment"},
				{"Supplier", "supplier"},
				{"Delivery Address", "delivery_address"},
				{"SKU", "sku"},
				{"Product", "product"},
				{"Received", "received"},
				{"Unit Price", "unit_price"},
				{"Amount", "amount"},
				{"Batch Plan", "batch_plan"},
				{"PO Date", "po_date"},
				{"Ordered at", "ordered_at"},
				{"Ordered by", "ordered_by"},
				{"Received at", "received_at"},
				{"Received by", "received_by"},
			},
		},
		// 5. imp_stb_ban
		{
			Table:    "imp_stb_ban",
			URL:      "https://dashboard.sampurna-group.com/api/public/card/3897a5a4-e48c-4def-9f21-acb9e35b0153/query/csv",
			Truncate: true,
			Columns: []colMapEntry{
				{"No. STB", "no_stb"},
				{"Received at", "received_at"},
				{"No. PO", "no_po"},
				{"Supplier", "supplier"},
				{"Delivery Address", "delivery_address"},
				{"SKU", "sku"},
				{"Product", "product"},
				{"Serial Number", "sn"},
			},
		},
		// 6. imp_pending_pr
		{
			Table:    "imp_pending_pr",
			URL:      "https://dashboard.sampurna-group.com/api/public/card/4960f161-115e-4c04-a489-6f5e150f6d78/query/csv",
			Truncate: true,
			Columns: []colMapEntry{
				{"Purchase Request", "pr_no"},
				{"Request Date", "tgl_request"},
				{"Purpose", "purpose"},
				{"Category", "kategori"},
				{"SKU", "sku"},
				{"Product", "produk"},
				{"Deliver to", "deliver_to"},
				{"Qty", "qty"},
				{"UoM", "uom"},
				{"Estimated Price", "estimated"},
				{"Total", "total"},
			},
		},
		// 7. ss_skb (Google Sheets: 1. DATA)
		{
			Table:      "ss_skb",
			URL:        "https://docs.google.com/spreadsheets/d/1Lss5tffHI5BxZWHwPIxJxK1RoACDea9vealJ6LfphrE/export?format=csv&gid=770515291",
			Truncate:   true,
			SkipRows:   2,
			IsByLetter: true,
			Columns: []colMapEntry{
				{"F", "tgl"},
				{"G", "sku"},
				{"I", "qty"},
				{"K", "nopol"},
				{"M", "program"},
				{"N", "statue"},
				{"O", "kategori"},
			},
		},
		// 8. ss_pok (Monitoring Ritase)
		{
			Table:    "ss_pok",
			URL:      "https://dashboard.sampurna-group.com/api/public/card/af0178f9-3515-43e9-a159-a3327272bad5/query/csv",
			Truncate: true,
			Columns: []colMapEntry{
				{"Nomor SJ", "sj"},
				{"Plat Nomor", "nopol"},
				{"Jenis Unit", "jenis_unit"},
				{"Sopir", "sopir"},
				{"Produk", "produk"},
				{"Asal", "src"},
				{"Customer", "cust"},
				{"Plant", "plant"},
				{"Volume Unit", "volume"},
				{"Qty Asal", "qty_src"},
				{"Satuan Asal", "satuan_src"},
				{"Qty Tujuan", "qty_dest"},
				{"Satuan Tujuan", "satuan_dest"},
				{"Tanggal SJ", "tgl_sj"},
				{"Tanggal UJT", "tgl_ujt"},
				{"Tanggal Terima Plant", "tgl_terima_plant"},
				{"Tanggal Pending", "tgl_pending"},
				{"Tanggal Close", "tgl_close"},
				{"Status", "statue"},
			},
		},
	}
}

func (s *csvSyncService) TriggerSync(table string) (bool, string) {
	s.state.Lock()
	if s.state.IsRunning {
		s.state.Unlock()
		return false, "Proses sinkronisasi CSV masih sedang berjalan. Silakan tunggu hingga selesai."
	}

	if s.db == nil {
		s.state.Unlock()
		return false, "Koneksi database CSV (imp_sampurna) belum tersedia."
	}

	// Cek batasan waktu: SELECT MAX(started) FROM log_imp (harus sudah lewat 5 menit / 300 detik)
	var diffSeconds *int
	err := s.db.Raw(`SELECT TIMESTAMPDIFF(SECOND, MAX(started), NOW()) FROM log_imp`).Scan(&diffSeconds).Error
	if err == nil && diffSeconds != nil && *diffSeconds >= 0 && *diffSeconds < 300 {
		s.state.Unlock()
		return false, "Last update less than 5 minutes, try again later."
	}

	now := time.Now().In(wibLoc)
	s.state.IsRunning = true
	s.state.StartedAt = &now
	s.state.FinishedAt = nil
	s.state.Status = "running"
	s.state.CurrentTable = table
	if table != "" && table != "all" {
		s.state.Message = fmt.Sprintf("Sedang menyinkronkan tabel %s...", table)
	} else {
		s.state.Message = "Sedang menyinkronkan seluruh tabel CSV..."
	}
	s.state.Logs = []string{fmt.Sprintf("[%s] Memulai proses sinkronisasi CSV (Native Go)...", now.Format("15:04:05"))}
	s.state.Unlock()

	go s.runImport(table)

	return true, "Proses sinkronisasi CSV berhasil dijalankan di background."
}

func (s *csvSyncService) appendLog(msg string, updateMsg bool) {
	s.state.Lock()
	defer s.state.Unlock()
	s.state.Logs = append(s.state.Logs, msg)
	if len(s.state.Logs) > 100 {
		s.state.Logs = s.state.Logs[len(s.state.Logs)-100:]
	}
	if updateMsg {
		s.state.Message = msg
	}
}

func (s *csvSyncService) runImport(filter string) {
	sqlDB, err := s.db.DB()
	if err != nil {
		s.finishWithError(fmt.Sprintf("Gagal mendapatkan koneksi SQL: %v", err))
		return
	}

	allItems := getImportConfigs()
	var toRun []struct {
		Idx int
		Cfg importItemConfig
	}

	filterTrim := strings.TrimSpace(filter)
	filterIdx, idxErr := strconv.Atoi(filterTrim)
	isNumericFilter := filterTrim != "" && idxErr == nil

	for idx, cfg := range allItems {
		if filterTrim != "" && filterTrim != "all" {
			if isNumericFilter && filterIdx != idx {
				continue
			}
			if !isNumericFilter && cfg.Table != filterTrim {
				continue
			}
		}
		toRun = append(toRun, struct {
			Idx int
			Cfg importItemConfig
		}{Idx: idx, Cfg: cfg})
	}

	if len(toRun) == 0 {
		s.finishWithError(fmt.Sprintf("Tabel '%s' tidak ditemukan dalam konfigurasi import.", filterTrim))
		return
	}

	s.appendLog("=== ASM Native Sync Service ===", false)
	s.appendLog(fmt.Sprintf("Menjalankan %d tugas sinkronisasi...", len(toRun)), false)

	overallSuccess := true
	var results []importResult

	for _, item := range toRun {
		s.appendLog(fmt.Sprintf("Import #%d: %s", item.Idx, item.Cfg.Table), true)
		res := s.importSingle(sqlDB, item.Cfg)
		results = append(results, res)
		if !res.Success {
			overallSuccess = false
		}
	}

	s.appendLog("=== SUMMARY ===", false)
	for _, r := range results {
		icon := "✓"
		if !r.Success {
			icon = "✗"
		}
		s.appendLog(fmt.Sprintf("%s %s: %d inserted, %d skipped", icon, r.Table, r.Inserted, r.Skipped), false)
		for _, e := range r.Errors {
			s.appendLog(fmt.Sprintf("   Error: %s", e), false)
		}
	}

	now := time.Now().In(wibLoc)
	s.state.Lock()
	defer s.state.Unlock()
	s.state.FinishedAt = &now
	s.state.IsRunning = false

	if overallSuccess {
		s.state.Status = "success"
		s.state.Message = "Sinkronisasi data CSV selesai dengan sukses!"
		s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] SINKRONISASI SELESAI DENGAN SUKSES.", now.Format("15:04:05")))
	} else {
		s.state.Status = "error"
		s.state.Message = "Sinkronisasi selesai dengan beberapa error."
		s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] ERROR: Terdapat tabel yang gagal disinkronkan.", now.Format("15:04:05")))
	}
}

func (s *csvSyncService) importSingle(sqlDB *sql.DB, cfg importItemConfig) importResult {
	res := importResult{Table: cfg.Table}

	var startedAt string
	if err := sqlDB.QueryRow("SELECT NOW()").Scan(&startedAt); err != nil || startedAt == "" {
		startedAt = time.Now().In(wibLoc).Format("2006-01-02 15:04:05")
	}

	// 1. Download CSV
	s.appendLog(fmt.Sprintf("Downloading CSV for [%s]...", cfg.Table), true)
	csvBytes, err := downloadCSV(cfg.URL)
	if err != nil || len(bytes.TrimSpace(csvBytes)) == 0 {
		errMsg := fmt.Sprintf("Failed to download CSV: %v", err)
		res.Errors = append(res.Errors, errMsg)
		s.appendLog("GAGAL download CSV: "+errMsg, false)
		return res
	}
	s.appendLog(fmt.Sprintf("Downloaded %d bytes", len(csvBytes)), false)

	// Hilangkan UTF-8 BOM jika ada
	csvBytes = bytes.TrimPrefix(csvBytes, []byte{0xEF, 0xBB, 0xBF})

	// 2. Parse CSV
	reader := csv.NewReader(bytes.NewReader(csvBytes))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var parsedRecords [][]string
	var headerIndexMap map[string]int

	if cfg.IsByLetter {
		rowNum := 0
		for {
			rec, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			rowNum++
			if rowNum <= cfg.SkipRows {
				continue
			}
			if len(rec) == 0 {
				continue
			}
			parsedRecords = append(parsedRecords, rec)
		}
		s.appendLog(fmt.Sprintf("Found %d rows (letter-based mapping)", len(parsedRecords)), false)
	} else {
		header, err := reader.Read()
		if err != nil || len(header) == 0 {
			errMsg := "CSV kosong atau header tidak valid"
			res.Errors = append(res.Errors, errMsg)
			s.appendLog(errMsg, false)
			return res
		}

		headerIndexMap = make(map[string]int, len(header))
		for i, colName := range header {
			cleanCol := strings.TrimSpace(colName)
			headerIndexMap[cleanCol] = i
		}

		var missingHeaders []string
		for _, c := range cfg.Columns {
			if _, ok := headerIndexMap[c.SourceKey]; !ok {
				missingHeaders = append(missingHeaders, c.SourceKey)
			}
		}
		if len(missingHeaders) > 0 {
			s.appendLog("WARNING: CSV header tidak ditemukan: "+strings.Join(missingHeaders, ", "), false)
		}

		for {
			rec, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			if len(rec) != len(header) {
				continue
			}
			parsedRecords = append(parsedRecords, rec)
		}
		s.appendLog(fmt.Sprintf("Found %d rows, %d columns", len(parsedRecords), len(header)), false)
	}

	if len(parsedRecords) == 0 {
		errMsg := "Data kosong setelah diparse"
		res.Errors = append(res.Errors, errMsg)
		s.appendLog(errMsg, false)
		return res
	}

	// 3. Truncate tabel jika dikonfigurasi
	if cfg.Truncate {
		s.appendLog(fmt.Sprintf("Truncating table `%s`...", cfg.Table), false)
		if _, err := sqlDB.Exec(fmt.Sprintf("TRUNCATE TABLE `%s`", cfg.Table)); err != nil {
			if _, err2 := sqlDB.Exec(fmt.Sprintf("DELETE FROM `%s`", cfg.Table)); err2 != nil {
				errMsg := fmt.Sprintf("Gagal truncate/delete table %s: %v", cfg.Table, err2)
				res.Errors = append(res.Errors, errMsg)
				s.appendLog(errMsg, false)
				return res
			}
			s.appendLog(fmt.Sprintf("Table `%s` cleared (DELETE)", cfg.Table), false)
		} else {
			s.appendLog(fmt.Sprintf("Table `%s` truncated", cfg.Table), false)
		}
	}

	// 4. Siapkan Prepared Statement
	mysqlCols := make([]string, len(cfg.Columns))
	placeholders := make([]string, len(cfg.Columns))
	for i, c := range cfg.Columns {
		mysqlCols[i] = fmt.Sprintf("`%s`", c.TargetCol)
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s)",
		cfg.Table,
		strings.Join(mysqlCols, ", "),
		strings.Join(placeholders, ", "),
	)

	// 5. Batch Insert dalam Transaction
	const batchSize = 500
	inserted := 0
	skipped := 0
	totalRows := len(parsedRecords)

	tx, err := sqlDB.Begin()
	if err != nil {
		errMsg := fmt.Sprintf("Gagal memulai transaksi: %v", err)
		res.Errors = append(res.Errors, errMsg)
		s.appendLog(errMsg, false)
		return res
	}

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		_ = tx.Rollback()
		errMsg := fmt.Sprintf("Gagal prepare query: %v", err)
		res.Errors = append(res.Errors, errMsg)
		s.appendLog(errMsg, false)
		return res
	}

	for rowIdx, rec := range parsedRecords {
		values := make([]any, len(cfg.Columns))
		nonEmpty := false

		for i, col := range cfg.Columns {
			rawVal := ""
			if cfg.IsByLetter {
				colIdx := colLetterToIndex(col.SourceKey)
				if colIdx >= 0 && colIdx < len(rec) {
					rawVal = strings.TrimSpace(rec[colIdx])
				}
			} else {
				if colIdx, ok := headerIndexMap[col.SourceKey]; ok && colIdx < len(rec) {
					rawVal = strings.TrimSpace(rec[colIdx])
				}
			}

			if rawVal != "" {
				nonEmpty = true
			}

			values[i] = formatColumnValue(cfg.Table, col.TargetCol, rawVal)
		}

		// Abaikan baris kosong total atau baris tanpa SKU pada ss_skb
		if !nonEmpty || (cfg.Table == "ss_skb" && strings.TrimSpace(fmt.Sprint(values[1])) == "") {
			skipped++
			continue
		}

		if _, err := stmt.Exec(values...); err != nil {
			skipped++
			if skipped <= 5 {
				s.appendLog(fmt.Sprintf("Row %d skipped: %v", rowIdx+1, err), false)
			}
			continue
		}
		inserted++

		if inserted > 0 && inserted%batchSize == 0 {
			_ = stmt.Close()
			if err := tx.Commit(); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Commit gagal: %v", err))
				s.appendLog(fmt.Sprintf("Commit gagal: %v", err), false)
				return res
			}
			tx, err = sqlDB.Begin()
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Begin ulang gagal: %v", err))
				return res
			}
			stmt, err = tx.Prepare(insertSQL)
			if err != nil {
				_ = tx.Rollback()
				res.Errors = append(res.Errors, fmt.Sprintf("Prepare ulang gagal: %v", err))
				return res
			}
			s.appendLog(fmt.Sprintf("Progress [%s]: %d / %d rows inserted...", cfg.Table, inserted, totalRows), true)
		}
	}

	_ = stmt.Close()
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		errMsg := fmt.Sprintf("Final commit gagal: %v", err)
		res.Errors = append(res.Errors, errMsg)
		s.appendLog(errMsg, false)
		return res
	}

	res.Success = true
	res.Inserted = inserted
	res.Skipped = skipped

	// 6. Catat log ke tabel log_imp
	if _, err := sqlDB.Exec("INSERT INTO `log_imp` (`namadata`, `jml`, `started`, `finished`) VALUES (?, ?, ?, NOW())", cfg.Table, inserted, startedAt); err != nil {
		s.appendLog(fmt.Sprintf("WARNING: Gagal simpan log ke `log_imp`: %v", err), false)
	} else {
		s.appendLog(fmt.Sprintf("DONE [%s]! Inserted: %d, Skipped: %d", cfg.Table, inserted, skipped), false)
	}

	return res
}

func formatColumnValue(table, targetCol, val string) any {
	switch {
	case targetCol == "tgl":
		if val == "" {
			return nil
		}
		return parseFlexibleDate(val)

	case isDateTimeTarget(targetCol):
		if val == "" {
			// imp_visitor.entrydate bertipe NOT NULL DEFAULT '0'
			if table == "imp_visitor" {
				return ""
			}
			return nil
		}
		cleaned := reTZOffset.ReplaceAllString(val, "")
		cleaned = strings.ReplaceAll(cleaned, "T", " ")
		return cleaned

	case targetCol == "tgl_sj":
		if val == "" {
			return nil
		}
		cleaned := reTZOffset.ReplaceAllString(val, "")
		if idx := strings.IndexByte(cleaned, 'T'); idx > 0 {
			cleaned = cleaned[:idx]
		}
		return cleaned

	case targetCol == "exitdate":
		if val != "" {
			cleaned := reTZOffset.ReplaceAllString(val, "")
			return strings.ReplaceAll(cleaned, "T", " ")
		}
		return ""

	case table == "ss_pok" && (targetCol == "volume" || targetCol == "qty_src" || targetCol == "qty_dest"):
		return parseFloatClean(val)

	case table == "imp_pending_pr" && (targetCol == "qty" || targetCol == "estimated" || targetCol == "total"):
		return int(math.Round(parseFloatClean(val)))

	case targetCol == "qty" || targetCol == "unit_price" || targetCol == "total_price" || targetCol == "received" || targetCol == "amount":
		return parseFloatClean(val)

	default:
		return val
	}
}

func isDateTimeTarget(col string) bool {
	switch col {
	case "entrydate", "released_at", "completed_at", "ordered_at", "received_at",
		"tgl_request", "tgl_ujt", "tgl_terima_plant", "tgl_pending", "tgl_close":
		return true
	}
	return false
}

func parseFloatClean(val string) float64 {
	if val == "" {
		return 0.0
	}
	cleaned := strings.ReplaceAll(val, ",", "")
	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0.0
	}
	return f
}

func parseFlexibleDate(val string) any {
	layouts := []string{
		"2 Jan 06",
		"02 Jan 06",
		"2 Jan 2006",
		"02 Jan 2006",
		"2006-01-02",
		"02/01/2006",
		"2/1/2006",
		"01/02/2006",
		"1/2/2006",
		"2006/01/02",
		"02-01-2006",
		"2-1-2006",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, val); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return nil
}

func colLetterToIndex(col string) int {
	col = strings.ToUpper(strings.TrimSpace(col))
	idx := 0
	for i := 0; i < len(col); i++ {
		idx = idx*26 + int(col[i]-'A'+1)
	}
	return idx - 1
}

func downloadCSV(rawURL string) ([]byte, error) {
	client := &http.Client{
		Timeout: 180 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/csv, */*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (s *csvSyncService) finishWithError(msg string) {
	s.state.Lock()
	defer s.state.Unlock()
	now := time.Now().In(wibLoc)
	s.state.IsRunning = false
	s.state.FinishedAt = &now
	s.state.Status = "error"
	s.state.Message = msg
	s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] ERROR: %s", now.Format("15:04:05"), msg))
}

func (s *csvSyncService) GetStatus() SyncStateSnapshot {
	s.state.Lock()
	defer s.state.Unlock()

	logsCopy := make([]string, len(s.state.Logs))
	copy(logsCopy, s.state.Logs)

	return SyncStateSnapshot{
		IsRunning:    s.state.IsRunning,
		StartedAt:    s.state.StartedAt,
		FinishedAt:   s.state.FinishedAt,
		Status:       s.state.Status,
		Message:      s.state.Message,
		CurrentTable: s.state.CurrentTable,
		Logs:         logsCopy,
	}
}
