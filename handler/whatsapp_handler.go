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

// Reconnect: Memaksa rekoneksi WebSocket bot WhatsApp
func (h *WhatsAppHandler) Reconnect(c *gin.Context) {
	err := h.bot.ForceReconnect()
	if c.Query("format") == "json" || c.GetHeader("Accept") == "application/json" {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Berhasil memicu rekoneksi WhatsApp"})
		return
	}
	c.Redirect(http.StatusSeeOther, "/wa/qr")
}

// Reset: Melakukan logout sesi dan reset ke mode QR baru
func (h *WhatsAppHandler) Reset(c *gin.Context) {
	err := h.bot.ResetSession()
	if c.Query("format") == "json" || c.GetHeader("Accept") == "application/json" {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Sesi WhatsApp berhasil direset"})
		return
	}
	c.Redirect(http.StatusSeeOther, "/wa/qr")
}

// GetQRImage: Mengembalikan binary PNG dari QR code WhatsApp
func (h *WhatsAppHandler) GetQRImage(c *gin.Context) {
	_, png, isLoggedIn := h.bot.GetQR()
	if isLoggedIn && h.bot.IsConnected() {
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
	connStatus, _ := status["status"].(string) // "CONNECTED" or "OFFLINE"
	phone, _ := status["phone"].(string)

	if isLoggedIn && connStatus == "CONNECTED" {
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html lang="id">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>WhatsApp Terhubung - ASM Bot</title>
				<script src="https://cdn.tailwindcss.com"></script>
				<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
			</head>
			<body class="bg-slate-100 flex items-center justify-center min-h-screen p-4 font-sans">
				<div class="bg-white p-8 rounded-2xl shadow-sm border border-slate-200 max-w-md w-full text-center space-y-5">
					<div class="w-16 h-16 bg-emerald-100 text-emerald-600 rounded-full flex items-center justify-center mx-auto text-3xl font-black">
						<i class="fa-solid fa-check"></i>
					</div>
					<div>
						<h1 class="text-xl font-bold text-slate-800">WhatsApp Sudah Terhubung!</h1>
						<p class="text-xs text-emerald-600 font-semibold mt-1">● Status: CONNECTED (Online)</p>
					</div>
					<div class="p-3 bg-slate-50 border border-slate-200 rounded-xl">
						<p class="text-xs text-slate-500">Nomor WhatsApp Aktif:</p>
						<p class="text-base text-slate-800 font-bold font-mono tracking-wide mt-0.5">%s</p>
					</div>
					<div class="p-3.5 bg-emerald-50 border border-emerald-100 rounded-xl text-xs text-emerald-800 leading-relaxed text-left space-y-1">
						<div class="font-bold flex items-center gap-1.5"><i class="fa-solid fa-robot"></i> Bot AutoResponder Aktif:</div>
						<div>Kirim perintah via WhatsApp: <code>/pendingharian</code>, <code>/spk</code>, <code>/omset</code>, dll.</div>
					</div>
					<div class="pt-2 flex items-center justify-center gap-3">
						<a href="/wa/reconnect" class="inline-flex items-center gap-1.5 px-3.5 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-lg transition border border-slate-200">
							<i class="fa-solid fa-rotate"></i> Reconnect
						</a>
						<a href="/wa/reset" onclick="return confirm('Apakah Anda yakin ingin logout dan scan QR ulang?')" class="inline-flex items-center gap-1.5 px-3.5 py-2 bg-rose-50 hover:bg-rose-100 text-rose-700 text-xs font-semibold rounded-lg transition border border-rose-200">
							<i class="fa-solid fa-right-from-bracket"></i> Logout & Scan Ulang
						</a>
					</div>
				</div>
			</body>
			</html>
		`, phone)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
		return
	}

	if isLoggedIn && connStatus != "CONNECTED" {
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html lang="id">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<meta http-equiv="refresh" content="5">
				<title>WhatsApp Terputus - ASM Bot</title>
				<script src="https://cdn.tailwindcss.com"></script>
				<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
			</head>
			<body class="bg-slate-100 flex items-center justify-center min-h-screen p-4 font-sans">
				<div class="bg-white p-8 rounded-2xl shadow-sm border border-amber-200 max-w-md w-full text-center space-y-5">
					<div class="w-16 h-16 bg-amber-100 text-amber-600 rounded-full flex items-center justify-center mx-auto text-3xl font-black animate-pulse">
						<i class="fa-solid fa-triangle-exclamation"></i>
					</div>
					<div>
						<h1 class="text-xl font-bold text-slate-800">WhatsApp Offline / Terputus</h1>
						<p class="text-xs text-amber-700 font-semibold mt-1">Sesi login tersimpan, tetapi koneksi socket terputus.</p>
					</div>
					<div class="p-3 bg-amber-50/70 border border-amber-200 rounded-xl">
						<p class="text-xs text-slate-500">Nomor Tertaut:</p>
						<p class="text-base text-slate-800 font-bold font-mono tracking-wide mt-0.5">%s</p>
					</div>
					<p class="text-xs text-slate-500 leading-relaxed">
						Watchdog otomatis mencoba menghubungkan kembali setiap 15 detik. Anda juga dapat menekan tombol di bawah untuk reconnect langsung.
					</p>
					<div class="pt-2 flex flex-col sm:flex-row items-center justify-center gap-2.5">
						<a href="/wa/reconnect" class="w-full sm:w-auto inline-flex items-center justify-center gap-1.5 px-4 py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold rounded-lg transition shadow-xs">
							<i class="fa-solid fa-rotate"></i> Hubungkan Ulang (Reconnect)
						</a>
						<a href="/wa/reset" onclick="return confirm('Apakah Anda yakin ingin logout dan scan QR ulang?')" class="w-full sm:w-auto inline-flex items-center justify-center gap-1.5 px-4 py-2.5 bg-rose-50 hover:bg-rose-100 text-rose-700 text-xs font-bold rounded-lg transition border border-rose-200">
							<i class="fa-solid fa-right-from-bracket"></i> Logout & Scan Ulang
						</a>
					</div>
					<p class="text-[11px] text-slate-400 italic">Halaman ini otomatis merefresh setiap 5 detik</p>
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
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Scan QR WhatsApp - ASM Bot</title>
			<meta http-equiv="refresh" content="5">
			<script src="https://cdn.tailwindcss.com"></script>
			<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
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
