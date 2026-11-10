package handler

import (
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
