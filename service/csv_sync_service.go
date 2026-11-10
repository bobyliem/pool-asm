package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func (s *csvSyncService) TriggerSync(table string) (bool, string) {
	s.state.Lock()
	if s.state.IsRunning {
		s.state.Unlock()
		return false, "Proses sinkronisasi CSV masih sedang berjalan. Silakan tunggu hingga selesai."
	}

	// Cek batasan waktu: SELECT MAX(started) FROM log_imp (harus sudah lewat 5 menit / 300 detik)
	if s.db != nil {
		var diffSeconds *int
		err := s.db.Raw(`SELECT TIMESTAMPDIFF(SECOND, MAX(started), NOW()) FROM log_imp`).Scan(&diffSeconds).Error
		if err == nil && diffSeconds != nil && *diffSeconds >= 0 && *diffSeconds < 300 {
			s.state.Unlock()
			return false, "Last update less than 5 minutes, try again later."
		}
	}

	phpPath := os.Getenv("PHP_PATH")
	if phpPath == "" {
		phpPath = "C:\\xampp\\php\\php.exe"
	}
	if _, err := os.Stat(phpPath); os.IsNotExist(err) {
		// Coba fallback ke PATH
		if p, err := exec.LookPath("php"); err == nil {
			phpPath = p
		} else {
			s.state.Unlock()
			return false, "PHP executable tidak ditemukan di C:\\xampp\\php\\php.exe maupun sistem PATH."
		}
	}

	sparkPath := os.Getenv("CI_SPARK_PATH")
	if sparkPath == "" {
		sparkPath = "C:\\xampp\\htdocs\\asm\\spark"
	}
	if _, err := os.Stat(sparkPath); os.IsNotExist(err) {
		s.state.Unlock()
		return false, fmt.Sprintf("Script Spark CodeIgniter tidak ditemukan di %s", sparkPath)
	}

	now := time.Now()
	s.state.IsRunning = true
	s.state.StartedAt = &now
	s.state.FinishedAt = nil
	s.state.Status = "running"
	s.state.CurrentTable = table
	if table != "" {
		s.state.Message = fmt.Sprintf("Sedang menyinkronkan tabel %s...", table)
	} else {
		s.state.Message = "Sedang menyinkronkan seluruh tabel CSV..."
	}
	s.state.Logs = []string{fmt.Sprintf("[%s] Memulai proses sinkronisasi CSV...", now.Format("15:04:05"))}
	s.state.Unlock()

	// Jalankan di background goroutine
	go s.runImport(phpPath, sparkPath, table)

	return true, "Proses sinkronisasi CSV berhasil dijalankan di background."
}

func (s *csvSyncService) runImport(phpPath, sparkPath, table string) {
	args := []string{sparkPath, "sync:data"}
	if table != "" && table != "all" {
		args = append(args, table)
	}

	cmd := exec.Command(phpPath, args...)
	cmd.Dir = filepath.Dir(sparkPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.finishWithError(fmt.Sprintf("Gagal membuka stdout pipe: %v", err))
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		s.finishWithError(fmt.Sprintf("Gagal menjalankan PHP: %v", err))
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		s.state.Lock()
		// Bersihkan ANSI color code jika ada
		cleanLine := stripAnsi(line)
		s.state.Logs = append(s.state.Logs, cleanLine)
		if len(s.state.Logs) > 100 {
			s.state.Logs = s.state.Logs[len(s.state.Logs)-100:]
		}

		if strings.Contains(cleanLine, "Import #") || strings.Contains(cleanLine, "Downloading CSV") {
			s.state.Message = cleanLine
		}
		s.state.Unlock()
	}

	cmdErr := cmd.Wait()

	s.state.Lock()
	defer s.state.Unlock()

	now := time.Now()
	s.state.FinishedAt = &now
	s.state.IsRunning = false

	if cmdErr != nil {
		s.state.Status = "error"
		s.state.Message = fmt.Sprintf("Sinkronisasi gagal: %v", cmdErr)
		s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] ERROR: %v", now.Format("15:04:05"), cmdErr))
	} else {
		s.state.Status = "success"
		s.state.Message = "Sinkronisasi data CSV selesai dengan sukses!"
		s.state.Logs = append(s.state.Logs, fmt.Sprintf("[%s] SINKRONISASI SELESAI DENGAN SUKSES.", now.Format("15:04:05")))
	}
}

func (s *csvSyncService) finishWithError(msg string) {
	s.state.Lock()
	defer s.state.Unlock()
	now := time.Now()
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

func stripAnsi(str string) string {
	// Simple cleanup of terminal escape codes
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(str); i++ {
		if str[i] == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if str[i] == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteByte(str[i])
	}
	return b.String()
}
