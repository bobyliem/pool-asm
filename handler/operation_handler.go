package handler

import (
	"net/http"

	"asm/service"
	"github.com/gin-gonic/gin"
)

type OperationHandler struct {
	operationService service.OperationService
}

func NewOperationHandler(svc service.OperationService) *OperationHandler {
	return &OperationHandler{operationService: svc}
}

func (h *OperationHandler) GetOperationSummary(c *gin.Context) {
	siteId := c.DefaultQuery("site_id", c.DefaultQuery("site", c.DefaultQuery("area", "ALL")))
	summary, err := h.operationService.GetOperationSummary(siteId)
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

func (h *OperationHandler) GetSites(c *gin.Context) {
	sites, err := h.operationService.GetSites()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   sites,
	})
}
