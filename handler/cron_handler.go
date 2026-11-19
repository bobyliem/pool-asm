package handler

import (
	"net/http"
	"strconv"

	"asm/models"
	"asm/service"

	"github.com/gin-gonic/gin"
)

type CronHandler struct {
	cronService service.CronService
}

func NewCronHandler(cronService service.CronService) *CronHandler {
	return &CronHandler{
		cronService: cronService,
	}
}

// GetJobs mengembalikan seluruh daftar cron job
func (h *CronHandler) GetJobs(c *gin.Context) {
	jobs, err := h.cronService.GetJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": jobs})
}

// ToggleJob mengaktifkan atau menonaktifkan job
func (h *CronHandler) ToggleJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	isActive, err := h.cronService.ToggleJob(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"message":   "Status job berhasil diubah",
		"is_active": isActive,
	})
}

// RunJob memicu eksekusi manual sebuah job
func (h *CronHandler) RunJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	job, err := h.cronService.GetJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Job tidak ditemukan"})
		return
	}

	started, msg := h.cronService.RunJob(job.JobKey)
	if !started {
		c.JSON(http.StatusConflict, gin.H{"status": "busy", "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": msg})
}

// UpdateJob memperbarui konfigurasi job
func (h *CronHandler) UpdateJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	var req models.UpdateCronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Payload JSON tidak valid"})
		return
	}

	if err := h.cronService.UpdateJob(id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Job berhasil diperbarui"})
}

// GetLogs mengambil riwayat eksekusi dari tabel zz_schedule_task dengan filter tanggal
func (h *CronHandler) GetLogs(c *gin.Context) {
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.cronService.GetLogs(tglAwal, tglAkhir, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": logs})
}
