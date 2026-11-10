package handler

import (
	"net/http"
	"time"

	"asm/service"

	"github.com/gin-gonic/gin"
)

type ProduksiHandler struct {
	produksiSvc service.ProduksiService
}

func NewProduksiHandler(svc service.ProduksiService) *ProduksiHandler {
	return &ProduksiHandler{produksiSvc: svc}
}

func (h *ProduksiHandler) GetResumePerHari(c *gin.Context) {
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	// Default ke hari ini jika parameter kosong
	now := time.Now()
	if tglAwal == "" {
		tglAwal = now.Format("2006-01-02")
	}
	if tglAkhir == "" {
		tglAkhir = now.Format("2006-01-02")
	}

	res, err := h.produksiSvc.GetResumePerHari(tglAwal, tglAkhir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data resume per hari: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   res,
	})
}

func (h *ProduksiHandler) GetSummaryPerDriver(c *gin.Context) {
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	// Default ke hari ini jika parameter kosong
	now := time.Now()
	if tglAwal == "" {
		tglAwal = now.Format("2006-01-02")
	}
	if tglAkhir == "" {
		tglAkhir = now.Format("2006-01-02")
	}

	res, err := h.produksiSvc.GetSummaryPerDriver(tglAwal, tglAkhir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data summary per driver: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   res,
	})
}

func (h *ProduksiHandler) GetMonitoringRitase(c *gin.Context) {
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	// Default ke hari ini jika parameter kosong
	now := time.Now()
	if tglAwal == "" {
		tglAwal = now.Format("2006-01-02")
	}
	if tglAkhir == "" {
		tglAkhir = now.Format("2006-01-02")
	}

	res, err := h.produksiSvc.GetMonitoringRitase(tglAwal, tglAkhir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data monitoring ritase: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   res,
	})
}
