package service

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"asm/models"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// WhatsAppNotifier interface untuk mengirim pesan WhatsApp tanpa cycle import
type WhatsAppNotifier interface {
	SendMessage(recipientPhone, text string) error
}

type CronService interface {
	Start()
	Stop()
	RunJob(jobKey string) (bool, string)
	GetJobs() ([]models.CronJob, error)
	GetJobByID(id int) (*models.CronJob, error)
	ToggleJob(id int) (bool, error)
	UpdateJob(id int, req models.UpdateCronJobRequest) error
	GetLogs(tglAwal, tglAkhir string, limit int) ([]models.ScheduleTaskLog, error)
}

type cronService struct {
	db               *gorm.DB
	csvSyncSvc       CsvSyncService
	solarSyncSvc     SolarSyncService
	stockpileSyncSvc StockpileSyncService
	waNotifier       WhatsAppNotifier

	cronParser cron.Parser
	wibLoc     *time.Location
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
}

func NewCronService(
	db *gorm.DB,
	csvSyncSvc CsvSyncService,
	solarSyncSvc SolarSyncService,
	stockpileSyncSvc StockpileSyncService,
	waNotifier WhatsAppNotifier,
) CronService {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	svc := &cronService{
		db:               db,
		csvSyncSvc:       csvSyncSvc,
		solarSyncSvc:     solarSyncSvc,
		stockpileSyncSvc: stockpileSyncSvc,
		waNotifier:       waNotifier,
		cronParser:       cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		wibLoc:           loc,
		stopChan:         make(chan struct{}),
	}

	// Auto seed 3 default jobs jika belum ada di tabel cron_jobs
	if db != nil {
		svc.seedDefaultJobs()
	}

	return svc
}

func (s *cronService) seedDefaultJobs() {
	defaults := []models.CronJob{
		{
			JobKey:   "sync_csv",
			Nama:     "CRON - CSV Sync",
			Desc:     "Sinkronisasi data Google Sheets / CSV ke database lokal",
			CronExpr: "0 6-23,0 * * *",
			IsActive: 1,
			NotifyWA: 1,
		},
		{
			JobKey:   "sync_solar",
			Nama:     "CRON - Solar Sync",
			Desc:     "Sinkronisasi pemakaian solar semua unit ke database lokal",
			CronExpr: "0 6-23,0 * * *",
			IsActive: 1,
			NotifyWA: 1,
		},
		{
			JobKey:   "sync_stockpile",
			Nama:     "CRON - Stockpile Sync",
			Desc:     "Sinkronisasi data stockpile dari Metabase ke database lokal",
			CronExpr: "0 6-23,0 * * *",
			IsActive: 1,
			NotifyWA: 1,
		},
	}

	for _, d := range defaults {
		var count int64
		s.db.Model(&models.CronJob{}).Where("job_key = ?", d.JobKey).Count(&count)
		if count == 0 {
			if err := s.db.Create(&d).Error; err != nil {
				log.Printf("[CronService] Warning seeding job %s: %v", d.JobKey, err)
			} else {
				log.Printf("[CronService] Berhasil seeding default job: %s (%s)", d.Nama, d.CronExpr)
			}
		}
	}
}

// Start memulai ticker background untuk mengevaluasi cron jobs tiap menit
func (s *cronService) Start() {
	if s.db == nil {
		log.Println("[CronService] Database tidak tersedia, scheduler tidak dijalankan.")
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Println("⏰ [CronService] In-App Scheduler started (Timezone: Asia/Jakarta)")

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastCheckedMin time.Time

		for {
			select {
			case <-s.stopChan:
				log.Println("🛑 [CronService] In-App Scheduler stopped.")
				return
			case t := <-ticker.C:
				now := t.In(s.wibLoc)
				currentMin := now.Truncate(time.Minute)

				if currentMin.Equal(lastCheckedMin) {
					continue
				}
				lastCheckedMin = currentMin

				s.checkAndTriggerJobs(currentMin)
			}
		}
	}()
}

func (s *cronService) Stop() {
	select {
	case <-s.stopChan:
		return
	default:
		close(s.stopChan)
		s.wg.Wait()
	}
}

// checkAndTriggerJobs memeriksa setiap job aktif di DB apakah cocok dengan waktu sekarang
func (s *cronService) checkAndTriggerJobs(currentMin time.Time) {
	var activeJobs []models.CronJob
	if err := s.db.Where("is_active = 1 AND is_running = 0").Find(&activeJobs).Error; err != nil {
		log.Printf("[CronService] Error query active jobs: %v", err)
		return
	}

	prevSec := currentMin.Add(-1 * time.Second)
	var matchedJobs []models.CronJob

	for _, job := range activeJobs {
		sched, err := s.cronParser.Parse(job.CronExpr)
		if err != nil {
			log.Printf("[CronService] Invalid cron expression for job %s (%s): %v", job.Nama, job.CronExpr, err)
			continue
		}

		nextTime := sched.Next(prevSec).Truncate(time.Minute)
		if nextTime.Equal(currentMin) {
			// Cegah double run di menit yang sama
			if job.LastRunAt != nil && job.LastRunAt.In(s.wibLoc).Truncate(time.Minute).Equal(currentMin) {
				continue
			}
			matchedJobs = append(matchedJobs, job)
		}
	}

	if len(matchedJobs) == 0 {
		return
	}

	log.Printf("⏰ [CronService] Menemukan %d job yang jadwalnya jatuh pada %s WIB", len(matchedJobs), currentMin.Format("15:04:05"))

	// Eksekusi job yang terjadwal secara sequential di background goroutine
	go func(jobs []models.CronJob) {
		for _, j := range jobs {
			s.executeJob(j.JobKey)
		}
	}(matchedJobs)
}

// RunJob memicu eksekusi manual sebuah job berdasarkan job_key
func (s *cronService) RunJob(jobKey string) (bool, string) {
	var job models.CronJob
	if err := s.db.Where("job_key = ?", jobKey).First(&job).Error; err != nil {
		return false, fmt.Sprintf("Job dengan key '%s' tidak ditemukan", jobKey)
	}

	if job.IsRunning == 1 {
		return false, fmt.Sprintf("Job '%s' saat ini masih sedang berjalan", job.Nama)
	}

	go s.executeJob(jobKey)
	return true, fmt.Sprintf("Job '%s' berhasil dipicu untuk dieksekusi", job.Nama)
}

// executeJob menjalankan 1 job dari awal sampai akhir, log ke zz_schedule_task, update cron_jobs, dan kirim WA
func (s *cronService) executeJob(jobKey string) {
	s.mu.Lock()
	var job models.CronJob
	if err := s.db.Where("job_key = ?", jobKey).First(&job).Error; err != nil {
		s.mu.Unlock()
		log.Printf("[CronService] Error find job %s: %v", jobKey, err)
		return
	}

	if job.IsRunning == 1 {
		s.mu.Unlock()
		log.Printf("[CronService] Job %s dilewati karena masih berstatus running", job.Nama)
		return
	}

	now := time.Now().In(s.wibLoc)

	// Tandai status running di tabel cron_jobs
	s.db.Model(&job).Updates(map[string]interface{}{
		"is_running":   1,
		"last_run_at":  now,
		"last_status":  "running",
		"last_message": "Job sedang berjalan...",
	})

	// 1. Masukkan log awal ke tabel zz_schedule_task
	taskLog := models.ScheduleTaskLog{
		Nama:        job.Nama,
		Flag:        0,
		Jml:         0,
		RunningTime: &now,
	}
	if err := s.db.Create(&taskLog).Error; err != nil {
		log.Printf("[CronService] Warning: Gagal insert ke zz_schedule_task: %v", err)
	}
	s.mu.Unlock()

	log.Printf("🚀 [CronService] Memulai eksekusi: %s (Task ID: %d)", job.Nama, taskLog.ID)

	var (
		success   bool
		jml       int
		resultMsg string
	)

	// 2. Eksekusi service terkait
	switch jobKey {
	case "sync_csv":
		if s.csvSyncSvc != nil {
			started, msg := s.csvSyncSvc.TriggerSync("all")
			if !started {
				success = false
				resultMsg = msg
			} else {
				success, jml, resultMsg = s.waitForCsvSync()
			}
		} else {
			success = false
			resultMsg = "CsvSyncService belum diinisialisasi"
		}

	case "sync_solar":
		if s.solarSyncSvc != nil {
			started, msg := s.solarSyncSvc.TriggerSync("all")
			if !started {
				success = false
				resultMsg = msg
			} else {
				success, jml, resultMsg = s.waitForSolarSync()
			}
		} else {
			success = false
			resultMsg = "SolarSyncService belum diinisialisasi"
		}

	case "sync_stockpile":
		if s.stockpileSyncSvc != nil {
			started, msg := s.stockpileSyncSvc.TriggerSync("", "")
			if !started {
				success = false
				resultMsg = msg
			} else {
				success, jml, resultMsg = s.waitForStockpileSync()
			}
		} else {
			success = false
			resultMsg = "StockpileSyncService belum diinisialisasi"
		}

	default:
		success = false
		resultMsg = fmt.Sprintf("Handler untuk job key '%s' tidak ditemukan", jobKey)
	}

	finishTime := time.Now().In(s.wibLoc)
	durationSec := int(finishTime.Sub(now).Seconds())

	statusStr := "failed"
	flagVal := 2
	if success {
		statusStr = "success"
		flagVal = 1
	}

	// 3. Update waktu finish, flag, dan jml di tabel zz_schedule_task
	if taskLog.ID > 0 {
		s.db.Model(&models.ScheduleTaskLog{}).Where("id = ?", taskLog.ID).Updates(map[string]interface{}{
			"finish_time": finishTime,
			"flag":        flagVal,
			"jml":         jml,
		})
	}

	// 4. Update status akhir di tabel cron_jobs
	s.db.Model(&models.CronJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"is_running":        0,
		"last_status":       statusStr,
		"last_duration_sec": durationSec,
		"last_message":      resultMsg,
	})

	log.Printf("🏁 [CronService] Selesai eksekusi: %s - Status: %s (%ds) - Info: %s", job.Nama, statusStr, durationSec, resultMsg)

	// 5. Kirim laporan ke WA admin di tabel settings
	if s.waNotifier != nil && job.NotifyWA == 1 {
		s.sendWhatsAppReport(job.Nama, now, finishTime, durationSec, statusStr, jml, resultMsg)
	}
}

func (s *cronService) waitForCsvSync() (bool, int, string) {
	time.Sleep(500 * time.Millisecond)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.After(20 * time.Minute)

	for {
		select {
		case <-timeout:
			return false, 0, "Timeout: Sinkronisasi CSV melebihi batas waktu 20 menit"
		case <-ticker.C:
			snap := s.csvSyncSvc.GetStatus()
			if !snap.IsRunning {
				return (snap.Status == "success"), 0, snap.Message
			}
		}
	}
}

func (s *cronService) waitForSolarSync() (bool, int, string) {
	time.Sleep(500 * time.Millisecond)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.After(20 * time.Minute)

	for {
		select {
		case <-timeout:
			return false, 0, "Timeout: Sinkronisasi Solar melebihi batas waktu 20 menit"
		case <-ticker.C:
			snap := s.solarSyncSvc.GetStatus()
			if !snap.IsRunning {
				return (snap.Status == "success"), snap.Total, snap.Message
			}
		}
	}
}

func (s *cronService) waitForStockpileSync() (bool, int, string) {
	time.Sleep(500 * time.Millisecond)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.After(20 * time.Minute)

	for {
		select {
		case <-timeout:
			return false, 0, "Timeout: Sinkronisasi Stockpile melebihi batas waktu 20 menit"
		case <-ticker.C:
			snap := s.stockpileSyncSvc.GetStatus()
			if !snap.IsRunning {
				return (snap.Status == "success"), snap.Total, snap.Message
			}
		}
	}
}

func (s *cronService) sendWhatsAppReport(jobNama string, startTime, finishTime time.Time, durationSec int, status string, jml int, message string) {
	var admins []struct {
		NoHp string `gorm:"column:no_hp"`
	}

	if err := s.db.Table("settings").
		Select("no_hp").
		Where("no_hp IS NOT NULL AND no_hp != '' AND no_hp != '0'").
		Find(&admins).Error; err != nil {
		log.Printf("[CronService] Error query settings: %v", err)
		return
	}

	if len(admins) == 0 {
		log.Println("[CronService] Tidak ada nomor WA di tabel settings untuk dikirimi laporan.")
		return
	}

	statusBadge := "✅ *SUKSES*"
	if status != "success" {
		statusBadge = "❌ *GAGAL*"
	}

	jmlLine := ""
	if jml > 0 {
		jmlLine = fmt.Sprintf("📦 *Jumlah Data:* %d baris\n", jml)
	}

	text := fmt.Sprintf(`🤖 *[CRON REPORT - ASM]*
━━━━━━━━━━━━━━━━━━━━
📌 *Job:* %s
⏱️ *Mulai:* %s WIB
🏁 *Selesai:* %s WIB (%ds)
📊 *Status:* %s
%s📝 *Keterangan:* %s
━━━━━━━━━━━━━━━━━━━━
_Sistem Otomatis ASM_`,
		jobNama,
		startTime.Format("02/01/2006 15:04:05"),
		finishTime.Format("02/01/2006 15:04:05"),
		durationSec,
		statusBadge,
		jmlLine,
		message,
	)

	for _, a := range admins {
		phone := strings.TrimSpace(a.NoHp)
		if phone != "" {
			go func(p string) {
				if err := s.waNotifier.SendMessage(p, text); err != nil {
					log.Printf("[CronService] Gagal kirim WA ke %s: %v", p, err)
				} else {
					log.Printf("[CronService] Berhasil kirim WA report ke %s", p)
				}
			}(phone)
		}
	}
}

func (s *cronService) GetJobs() ([]models.CronJob, error) {
	var jobs []models.CronJob
	err := s.db.Order("id ASC").Find(&jobs).Error
	return jobs, err
}

func (s *cronService) GetJobByID(id int) (*models.CronJob, error) {
	var job models.CronJob
	err := s.db.First(&job, id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *cronService) ToggleJob(id int) (bool, error) {
	var job models.CronJob
	if err := s.db.First(&job, id).Error; err != nil {
		return false, err
	}

	newStatus := 0
	if job.IsActive == 0 {
		newStatus = 1
	}

	err := s.db.Model(&job).Update("is_active", newStatus).Error
	return newStatus == 1, err
}

func (s *cronService) UpdateJob(id int, req models.UpdateCronJobRequest) error {
	var job models.CronJob
	if err := s.db.First(&job, id).Error; err != nil {
		return err
	}

	updates := make(map[string]interface{})
	if req.Nama != "" {
		updates["nama"] = req.Nama
	}
	if req.Desc != "" {
		updates["desc"] = req.Desc
	}
	if req.CronExpr != "" {
		if _, err := s.cronParser.Parse(req.CronExpr); err != nil {
			return fmt.Errorf("format cron expression '%s' tidak valid: %v", req.CronExpr, err)
		}
		updates["cron_expr"] = req.CronExpr
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.NotifyWA != nil {
		updates["notify_wa"] = *req.NotifyWA
	}

	return s.db.Model(&job).Updates(updates).Error
}

func (s *cronService) GetLogs(tglAwal, tglAkhir string, limit int) ([]models.ScheduleTaskLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var logs []models.ScheduleTaskLog
	query := s.db.Model(&models.ScheduleTaskLog{})

	if tglAwal != "" {
		query = query.Where("DATE(running_time) >= ?", tglAwal)
	}
	if tglAkhir != "" {
		query = query.Where("DATE(running_time) <= ?", tglAkhir)
	}

	err := query.Order("id DESC").Limit(limit).Find(&logs).Error
	return logs, err
}
