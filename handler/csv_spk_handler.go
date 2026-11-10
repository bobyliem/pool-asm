package handler

import (
	"net/http"
	"time"

	"asm/models"
	"asm/service"

	"github.com/gin-gonic/gin"
)

type CsvSpkHandler struct {
	spkService service.CsvSpkService
}

func NewCsvSpkHandler(svc service.CsvSpkService) *CsvSpkHandler {
	return &CsvSpkHandler{spkService: svc}
}

func (h *CsvSpkHandler) GetFilterOptions(c *gin.Context) {
	options, err := h.spkService.GetFilterOptions()
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

func (h *CsvSpkHandler) GetSpkPerCust(c *gin.Context) {
	area := c.Query("area")
	cust := c.Query("cust")
	purpose := c.Query("purpose")
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	// Default ke bulan ini jika kosong
	if tglAwal == "" {
		tglAwal = time.Now().Format("2006-01") + "-01"
	}
	if tglAkhir == "" {
		tglAkhir = time.Now().Format("2006-01-02")
	}

	data, err := h.spkService.GetSpkPerCust(area, cust, purpose, tglAwal, tglAkhir)
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

func (h *CsvSpkHandler) GetSpkDetail(c *gin.Context) {
	nopol := c.Query("nopol")
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	if nopol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter nopol wajib diisi",
		})
		return
	}

	if tglAwal == "" {
		tglAwal = time.Now().Format("2006-01") + "-01"
	}
	if tglAkhir == "" {
		tglAkhir = time.Now().Format("2006-01-02")
	}

	data, err := h.spkService.GetSpkDetail(nopol, tglAwal, tglAkhir)
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

func (h *CsvSpkHandler) GetSummaryPerCust(c *gin.Context) {
	area := c.Query("area")
	cust := c.Query("cust")
	purpose := c.Query("purpose")
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	if tglAwal == "" {
		tglAwal = time.Now().Format("2006-01") + "-01"
	}
	if tglAkhir == "" {
		tglAkhir = time.Now().Format("2006-01-02")
	}

	data, err := h.spkService.GetSummaryPerCust(area, cust, purpose, tglAwal, tglAkhir)
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

func (h *CsvSpkHandler) GetResumePerKategori(c *gin.Context) {
	area := c.Query("area")
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	if tglAwal == "" {
		tglAwal = time.Now().Format("2006-01") + "-01"
	}
	if tglAkhir == "" {
		tglAkhir = time.Now().Format("2006-01-02")
	}

	data, err := h.spkService.GetResumePerKategori(area, tglAwal, tglAkhir)
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

func (h *CsvSpkHandler) GetResumeKategoriDetail(c *gin.Context) {
	area := c.Query("area")
	cust := c.Query("cust")
	category := c.Query("category")
	status := c.Query("status")
	tglAwal := c.Query("tglawal")
	tglAkhir := c.Query("tglakhir")

	if tglAwal == "" {
		tglAwal = time.Now().Format("2006-01") + "-01"
	}
	if tglAkhir == "" {
		tglAkhir = time.Now().Format("2006-01-02")
	}

	items, err := h.spkService.GetResumeKategoriDetail(area, cust, category, status, tglAwal, tglAkhir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": models.CsvResumeKategoriDetailResponse{
			Customer: cust,
			Status:   status,
			Category: category,
			TglAwal:  tglAwal,
			TglAkhir: tglAkhir,
			Items:    items,
		},
	})
}
