package handler

import (
	"fmt"
	"net/http"

	"asm/models"
	"asm/service"

	"github.com/gin-gonic/gin"
)

type SolarHandler struct {
	service     service.SolarService
	syncService service.SolarSyncService
}

func NewSolarHandler(service service.SolarService, syncService service.SolarSyncService) *SolarHandler {
	return &SolarHandler{
		service:     service,
		syncService: syncService,
	}
}

// GetPemakaianSolar handles GET /api/v1/solar/pemakaian
func (h *SolarHandler) GetPemakaianSolar(c *gin.Context) {
	var filter models.PemakaianSolarFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter filter tidak valid: " + err.Error(),
		})
		return
	}

	result, err := h.service.GetPemakaianSolar(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data pemakaian solar: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result["data"],
		"meta": gin.H{
			"total_rows": result["total_rows"],
			"page":       result["page"],
			"limit":      result["limit"],
		},
		"summary": result["summary"],
	})
}

// GetFilterOptions handles GET /api/v1/solar/filter-options
func (h *SolarHandler) GetFilterOptions(c *gin.Context) {
	options, err := h.service.GetFilterOptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil opsi filter: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   options,
	})
}

// TriggerSync handles POST /api/v1/solar/sync
func (h *SolarHandler) TriggerSync(c *gin.Context) {
	var req struct {
		Unit string `json:"unit" form:"unit"`
	}
	_ = c.ShouldBind(&req)

	if req.Unit == "" {
		req.Unit = c.Query("unit")
	}

	success, msg := h.syncService.TriggerSync(req.Unit)
	if !success {
		c.JSON(http.StatusConflict, gin.H{
			"status":  "warning",
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": msg,
	})
}

// GetSyncStatus handles GET /api/v1/solar/sync/status
func (h *SolarHandler) GetSyncStatus(c *gin.Context) {
	status := h.syncService.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   status,
	})
}

// GetDtSolar handles GET /api/v1/solar/dt-solar
func (h *SolarHandler) GetDtSolar(c *gin.Context) {
	var filter models.DtSolarFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter filter tidak valid: " + err.Error(),
		})
		return
	}

	result, err := h.service.GetDtSolar(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data dt_solar: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result["data"],
		"meta": gin.H{
			"total_rows": result["total_rows"],
			"page":       result["page"],
			"limit":      result["limit"],
		},
		"summary": result["summary"],
	})
}

// GetDtSolarFilterOptions handles GET /api/v1/solar/dt-solar/filters
func (h *SolarHandler) GetDtSolarFilterOptions(c *gin.Context) {
	options, err := h.service.GetDtSolarFilterOptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil opsi filter: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   options,
	})
}

// UploadDtSolar handles POST /api/v1/solar/dt-solar/upload
func (h *SolarHandler) UploadDtSolar(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "File tidak ditemukan atau format upload salah: " + err.Error(),
		})
		return
	}
	defer file.Close()

	defaultUnit := c.PostForm("unit")

	sourceRows, importedRows, err := h.service.ImportDtSolarFromReader(file, header.Filename, defaultUnit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal import data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Berhasil mengimpor %d baris sumber -> %d baris data solar tersimpan dari file %s", sourceRows, importedRows, header.Filename),
		"data": gin.H{
			"source_rows":   sourceRows,
			"imported_rows": importedRows,
		},
	})
}

// ImportSampleDtSolar handles POST /api/v1/solar/dt-solar/import-sample
func (h *SolarHandler) ImportSampleDtSolar(c *gin.Context) {
	sourceRows, importedRows, err := h.service.ImportDtSolarFromSample()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal import sample data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Berhasil mengimpor sample data: %d baris sumber -> %d baris data solar tersimpan", sourceRows, importedRows),
		"data": gin.H{
			"source_rows":   sourceRows,
			"imported_rows": importedRows,
		},
	})
}

// TruncateDtSolar handles DELETE /api/v1/solar/dt-solar/truncate
func (h *SolarHandler) TruncateDtSolar(c *gin.Context) {
	if err := h.service.TruncateDtSolar(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengosongkan data dt_solar: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Semua data di dt_solar berhasil dikosongkan",
	})
}

// ==========================================
// LAPORAN COLOKAN SOLAR HANDLERS
// ==========================================

// GetLaporanDetailHarian handles GET /api/v1/solar/laporan/detail-harian
func (h *SolarHandler) GetLaporanDetailHarian(c *gin.Context) {
	var filter models.LaporanDetailHarianFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter filter tidak valid: " + err.Error(),
		})
		return
	}

	result, err := h.service.GetLaporanDetailHarian(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data detail harian: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result["data"],
		"meta": gin.H{
			"total_rows": result["total_rows"],
			"page":       result["page"],
			"limit":      result["limit"],
		},
		"summary": result["summary"],
	})
}

// GetLaporanRekapUnit handles GET /api/v1/solar/laporan/rekap-unit
func (h *SolarHandler) GetLaporanRekapUnit(c *gin.Context) {
	var filter models.LaporanRekapUnitFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter filter tidak valid: " + err.Error(),
		})
		return
	}

	result, err := h.service.GetLaporanRekapUnit(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data rekap unit: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    result["data"],
		"summary": result["summary"],
	})
}

// GetLaporanRekonsiliasi handles GET /api/v1/solar/laporan/rekonsiliasi
func (h *SolarHandler) GetLaporanRekonsiliasi(c *gin.Context) {
	var filter models.LaporanRekonsiliasiFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter filter tidak valid: " + err.Error(),
		})
		return
	}

	result, err := h.service.GetLaporanRekonsiliasi(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data rekonsiliasi & anomali: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result["data"],
		"meta": gin.H{
			"total_rows": result["total_rows"],
			"page":       result["page"],
			"limit":      result["limit"],
		},
		"summary": result["summary"],
	})
}

// GetDetailPoints handles GET /api/v1/solar/laporan/detail-points
func (h *SolarHandler) GetDetailPoints(c *gin.Context) {
	tgl := c.Query("tgl")
	unit := c.Query("unit")
	if tgl == "" || unit == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter tgl dan unit wajib diisi",
		})
		return
	}

	points, err := h.service.GetDetailPoints(tgl, unit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil detail points: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   points,
	})
}

// GetLaporanFilterOptions handles GET /api/v1/solar/laporan/filters
func (h *SolarHandler) GetLaporanFilterOptions(c *gin.Context) {
	options, err := h.service.GetLaporanFilterOptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil filter options: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   options,
	})
}
