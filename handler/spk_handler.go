package handler

import (
	"net/http"
	"time"

	"asm/service"
	"github.com/gin-gonic/gin"
)

type SpkHandler struct {
	spkService service.SpkService
}

func NewSpkHandler(svc service.SpkService) *SpkHandler {
	return &SpkHandler{spkService: svc}
}

func (h *SpkHandler) GetSpkPerCust(c *gin.Context) {
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")
	area := c.Query("area")
	cust := c.Query("cust")
	purpose := c.Query("purpose")

	if tglAwal == "" {
		tglAwal = time.Now().Format("2006-01-02")
	}
	if tglAkhir == "" {
		tglAkhir = time.Now().Format("2006-01-02")
	}

	data, err := h.spkService.GetSpkPerCust(tglAwal, tglAkhir, area, cust, purpose)
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

func (h *SpkHandler) GetFilterOptions(c *gin.Context) {
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")
	area := c.Query("area")

	if tglAwal == "" {
		tglAwal = time.Now().Format("2006-01-02")
	}
	if tglAkhir == "" {
		tglAkhir = time.Now().Format("2006-01-02")
	}

	options, err := h.spkService.GetFilterOptions(tglAwal, tglAkhir, area)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   options,
	})
}

