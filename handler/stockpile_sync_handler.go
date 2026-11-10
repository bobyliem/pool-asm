package handler

import (
	"net/http"

	"asm/service"

	"github.com/gin-gonic/gin"
)

type StockpileSyncHandler struct {
	syncService service.StockpileSyncService
}

func NewStockpileSyncHandler(syncService service.StockpileSyncService) *StockpileSyncHandler {
	return &StockpileSyncHandler{
		syncService: syncService,
	}
}

type StockpileSyncRequest struct {
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`
}

// TriggerSync handles POST /api/v1/stockpile/sync
// Parameter opsional: start_date & end_date (format YYYY-MM-DD).
// Jika tidak disertakan, default adalah 7 hari yang lalu s/d hari ini.
func (h *StockpileSyncHandler) TriggerSync(c *gin.Context) {
	var req StockpileSyncRequest
	_ = c.ShouldBind(&req)

	if req.StartDate == "" {
		req.StartDate = c.Query("start_date")
	}
	if req.EndDate == "" {
		req.EndDate = c.Query("end_date")
	}

	success, msg := h.syncService.TriggerSync(req.StartDate, req.EndDate)
	if !success {
		c.JSON(http.StatusConflict, gin.H{
			"status":  "busy",
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "started",
		"message": msg,
	})
}

// GetSyncStatus handles GET /api/v1/stockpile/sync/status
func (h *StockpileSyncHandler) GetSyncStatus(c *gin.Context) {
	snapshot := h.syncService.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   snapshot,
	})
}
