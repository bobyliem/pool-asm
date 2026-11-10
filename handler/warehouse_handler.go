package handler

import (
	"net/http"

	"asm/service"

	"github.com/gin-gonic/gin"
)

type WarehouseHandler struct {
	svc service.WarehouseService
}

func NewWarehouseHandler(svc service.WarehouseService) *WarehouseHandler {
	return &WarehouseHandler{svc: svc}
}

func (h *WarehouseHandler) GetCriticalInventory(c *gin.Context) {
	site := c.Query("site")

	resp, err := h.svc.GetCriticalInventory(site)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data critical inventory: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *WarehouseHandler) GetSites(c *gin.Context) {
	sites, err := h.svc.GetSites()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil daftar site: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   sites,
	})
}
