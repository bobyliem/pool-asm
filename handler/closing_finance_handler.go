package handler

import (
	"asm/models"
	"asm/service"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type ClosingFinanceHandler struct {
	service service.ClosingFinanceService
}

func NewClosingFinanceHandler(s service.ClosingFinanceService) *ClosingFinanceHandler {
	return &ClosingFinanceHandler{service: s}
}

func (h *ClosingFinanceHandler) GetData(c *gin.Context) {
	var filter models.ClosingFinanceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter tidak valid: " + err.Error(),
		})
		return
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}

	list, total, summary, err := h.service.GetData(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data closing finance: " + err.Error(),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"data":        list,
		"total":       total,
		"page":        filter.Page,
		"limit":       filter.Limit,
		"total_pages": totalPages,
		"summary":     summary,
	})
}

func (h *ClosingFinanceHandler) GetFilterOptions(c *gin.Context) {
	options, err := h.service.GetFilterOptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil opsi filter: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   options,
	})
}

func (h *ClosingFinanceHandler) UploadExcel(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "File tidak ditemukan atau format upload salah",
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Hanya file berekstensi .xlsx atau .xls yang diperbolehkan",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal membuka file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	totalImported, err := h.service.ProcessExcelUpload(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"message":        fmt.Sprintf("Tabel ext_closing berhasil di-truncate dan diisi kembali dengan %d baris data.", totalImported),
		"total_imported": totalImported,
	})
}

func (h *ClosingFinanceHandler) Truncate(c *gin.Context) {
	if err := h.service.Truncate(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengosongkan tabel: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Tabel ext_closing berhasil dikosongkan.",
	})
}
