package handler

import (
	"net/http"

	"asm/service"
	"github.com/gin-gonic/gin"
)

type HoDashboardHandler struct {
	dashboardService service.DashboardService
}

func NewHoDashboardHandler(svc service.DashboardService) *HoDashboardHandler {
	return &HoDashboardHandler{dashboardService: svc}
}

func (h *HoDashboardHandler) GetVisitorSummary(c *gin.Context) {
	summary, err := h.dashboardService.GetDashboardVisitorSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   summary,
	})
}

