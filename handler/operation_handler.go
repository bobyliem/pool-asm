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
	summary, err := h.operationService.GetOperationSummary()
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

