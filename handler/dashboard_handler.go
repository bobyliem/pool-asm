package handler

import (
	"net/http"

	"asm/service"

	"github.com/gin-gonic/gin"
)

type CsvDashboardHandler struct {
	service service.CsvDashboardService
}

func NewCsvDashboardHandler(service service.CsvDashboardService) *CsvDashboardHandler {
	return &CsvDashboardHandler{service: service}
}

func (h *CsvDashboardHandler) GetVisitorSummary(c *gin.Context) {
	summary, err := h.service.GetVisitorSummary()
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

func (h *CsvDashboardHandler) GetAreas(c *gin.Context) {
	areas, err := h.service.GetAreas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   areas,
	})
}

func (h *CsvDashboardHandler) GetCategoryDashboardData(c *gin.Context) {
	area := c.DefaultQuery("area", "ALL")

	data, err := h.service.GetCategoryDashboardData(area)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
	})
}
