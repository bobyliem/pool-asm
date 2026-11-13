package handler

import (
	"fmt"
	"net/http"

	"asm/service/whatsapp"

	"github.com/gin-gonic/gin"
)

type WhatsAppHandler struct {
	bot *whatsapp.WhatsAppBot
}

func NewWhatsAppHandler(bot *whatsapp.WhatsAppBot) *WhatsAppHandler {
	return &WhatsAppHandler{bot: bot}
}

// GetStatus: Menampilkan status koneksi WhatsApp
func (h *WhatsAppHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   h.bot.GetStatus(),
	})
}

// GetQRImage: Mengembalikan binary PNG dari QR code WhatsApp
func (h *WhatsAppHandler) GetQRImage(c *gin.Context) {
	_, png, isLoggedIn := h.bot.GetQR()
	if isLoggedIn {
		c.String(http.StatusOK, "WhatsApp sudah terhubung.")
		return
	}
	if len(png) == 0 {
		c.String(http.StatusNotFound, "QR code belum tersedia. Silakan refresh sebentar lagi.")
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

// ShowQRPage: Menampilkan halaman web bersih dengan auto-refresh untuk Scan QR WhatsApp
func (h *WhatsAppHandler) ShowQRPage(c *gin.Context) {
	status := h.bot.GetStatus()
	isLoggedIn, _ := status["is_logged_in"].(bool)
	phone, _ := status["phone"].(string)

	if isLoggedIn {
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html lang="id">
			<head>
				<meta charset="UTF-8">
				<title>WhatsApp Terhubung</title>
				<script src="https://cdn.tailwindcss.com"></script>
			</head>
			<body class="bg-slate-100 flex items-center justify-center min-h-screen p-4 font-sans">
				<div class="bg-white p-8 rounded-2xl shadow-sm border border-slate-200 max-w-md w-full text-center space-y-4">
					<div class="w-16 h-16 bg-emerald-100 text-emerald-600 rounded-full flex items-center justify-center mx-auto text-3xl font-black">
						✓
					</div>
					<h1 class="text-xl font-bold text-slate-800">WhatsApp Sudah Terhubung!</h1>
					<p class="text-sm text-slate-500">Nomor Aktif: <strong class="text-slate-800 font-mono">%s</strong></p>
					<div class="p-3 bg-emerald-50 rounded-lg text-xs text-emerald-700 font-medium">
						Bot AutoResponder siap menerima perintah seperti /spk, /nopol, /omset, dll.
					</div>
				</div>
			</body>
			</html>
		`, phone)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
		return
	}

	html := `
		<!DOCTYPE html>
		<html lang="id">
		<head>
			<meta charset="UTF-8">
			<title>Scan QR WhatsApp - ASM Bot</title>
			<meta http-equiv="refresh" content="5">
			<script src="https://cdn.tailwindcss.com"></script>
		</head>
		<body class="bg-slate-100 flex items-center justify-center min-h-screen p-4 font-sans">
			<div class="bg-white p-8 rounded-2xl shadow-sm border border-slate-200 max-w-md w-full text-center space-y-4">
				<h1 class="text-xl font-bold text-slate-800">Scan QR Code WhatsApp</h1>
				<p class="text-xs text-slate-500">Buka WhatsApp di HP ➔ Titik 3 / Setelan ➔ <strong>Perangkat Tertaut</strong> ➔ <strong>Tautkan Perangkat</strong></p>
				<div class="p-4 bg-slate-50 rounded-xl border border-slate-200 inline-block">
					<img src="/wa/qr/image" alt="QR Code WhatsApp" class="w-64 h-64 mx-auto rounded-lg shadow-xs" onerror="this.src=''; this.alt='Memuat QR Code...';">
				</div>
				<p class="text-[11px] text-slate-400 italic">Halaman ini otomatis merefresh setiap 5 detik</p>
			</div>
		</body>
		</html>
	`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// SendMessage: Endpoint API untuk mengirim pesan WhatsApp dari sistem lain
func (h *WhatsAppHandler) SendMessage(c *gin.Context) {
	var req struct {
		Phone   string `json:"phone" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Parameter phone dan message wajib diisi",
		})
		return
	}

	err := h.bot.SendMessage(req.Phone, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Pesan berhasil dikirim",
	})
}
