package handler

import (
	"net/http"

	"asm/service"

	"github.com/gin-gonic/gin"
)

type CsvSyncHandler struct {
	syncService service.CsvSyncService
}

func NewCsvSyncHandler(syncService service.CsvSyncService) *CsvSyncHandler {
	return &CsvSyncHandler{
		syncService: syncService,
	}
}

type SyncRequest struct {
	Table string `json:"table" form:"table"`
}

func (h *CsvSyncHandler) TriggerSync(c *gin.Context) {
	var req SyncRequest
	if err := c.ShouldBind(&req); err != nil {
		req.Table = c.Query("table")
	}
	if req.Table == "" {
		req.Table = c.Query("table")
	}

	success, msg := h.syncService.TriggerSync(req.Table)
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
		"table":   req.Table,
	})
}

func (h *CsvSyncHandler) GetSyncStatus(c *gin.Context) {
	snapshot := h.syncService.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   snapshot,
	})
}
