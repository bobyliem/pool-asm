package handler

import (
	"encoding/base64"
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

	_, qrPNG, _ := h.bot.GetQR()

	var qrElement string
	if len(qrPNG) > 0 {
		qrBase64 := base64.StdEncoding.EncodeToString(qrPNG)
		qrElement = fmt.Sprintf(`
				<div class="p-3 bg-white rounded-xl border border-slate-200 inline-block shadow-xs">
					<img id="qr-image" src="data:image/png;base64,%s" alt="QR Code WhatsApp" class="w-60 h-60 mx-auto rounded-lg">
				</div>
				<div class="pt-2 flex items-center justify-center gap-2">
					<button onclick="requestNewQR()" class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-lg transition border border-slate-200">
						<i class="fa-solid fa-rotate"></i> Refresh QR Code
					</button>
				</div>
		`, qrBase64)
	} else {
		qrElement = `
				<div class="p-8 bg-slate-50 rounded-xl border border-dashed border-slate-300 w-60 h-60 mx-auto flex flex-col items-center justify-center space-y-3">
					<i class="fa-solid fa-circle-notch fa-spin text-3xl text-emerald-600"></i>
					<p class="text-xs text-slate-700 font-semibold">Menyiapkan QR Code...</p>
					<p class="text-[11px] text-slate-400">Sedang menghubungi server WhatsApp</p>
				</div>
				<div class="pt-2 flex items-center justify-center gap-2">
					<button onclick="requestNewQR()" class="inline-flex items-center gap-1.5 px-3.5 py-2 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold rounded-lg transition shadow-xs">
						<i class="fa-solid fa-rotate"></i> Klik untuk Muat QR Baru
					</button>
				</div>
		`
	}

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="id">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Tautkan WhatsApp - ASM Bot</title>
			<script src="https://cdn.tailwindcss.com"></script>
			<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
		</head>
		<body class="bg-slate-100 flex items-center justify-center min-h-screen p-4 font-sans">
			<div class="bg-white p-6 sm:p-8 rounded-2xl shadow-sm border border-slate-200 max-w-lg w-full text-center space-y-5">
				<div>
					<div class="w-12 h-12 bg-emerald-100 text-emerald-600 rounded-full flex items-center justify-center mx-auto text-2xl font-bold mb-3">
						<i class="fa-brands fa-whatsapp"></i>
					</div>
					<h1 class="text-xl font-bold text-slate-800">Tautkan Perangkat WhatsApp</h1>
					<p class="text-xs text-slate-500 mt-1">Pilih salah satu metode di bawah untuk menghubungkan bot:</p>
				</div>

				<!-- Tab Navigation -->
				<div class="flex rounded-xl bg-slate-100 p-1 border border-slate-200 text-xs font-bold">
					<button id="tab-qr-btn" onclick="switchTab('qr')" class="flex-1 py-2 rounded-lg bg-white text-emerald-700 shadow-xs transition flex items-center justify-center gap-1.5">
						<i class="fa-solid fa-qrcode"></i> Metode 1: Scan QR
					</button>
					<button id="tab-code-btn" onclick="switchTab('code')" class="flex-1 py-2 rounded-lg text-slate-600 hover:text-slate-800 transition flex items-center justify-center gap-1.5">
						<i class="fa-solid fa-key"></i> Metode 2: Kode Telepon
					</button>
				</div>

				<!-- Tab 1: QR Code -->
				<div id="tab-qr" class="space-y-3">
					<p class="text-xs text-slate-500">
						Buka WA di HP ➔ Setelan / Titik Tiga ➔ <strong>Perangkat Tertaut</strong> ➔ <strong>Tautkan Perangkat</strong>
					</p>
					<div id="qr-container">
						%s
					</div>
					<p class="text-[11px] text-slate-400 italic">QR otomatis diperbarui secara realtime</p>
				</div>

				<!-- Tab 2: Pairing Code via Phone -->
				<div id="tab-code" class="hidden space-y-4 text-left">
					<div class="p-3.5 bg-blue-50/70 border border-blue-200 rounded-xl text-xs text-blue-900 leading-relaxed space-y-1">
						<div class="font-bold flex items-center gap-1.5"><i class="fa-solid fa-circle-info"></i> Tautkan Tanpa Kamera (Paling Stabil):</div>
						<p>Jika scan QR gagal atau kamera bermasalah, gunakan kode 8-karakter ini langsung dari WhatsApp HP Anda.</p>
					</div>

					<div class="space-y-1.5">
						<label class="block text-xs font-bold text-slate-700">Nomor WhatsApp HP Anda:</label>
						<div class="flex gap-2">
							<input type="text" id="phone-input" placeholder="08123456789 atau 628123456789" class="flex-1 px-3.5 py-2.5 text-sm border border-slate-300 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:outline-none font-mono">
							<button id="btn-get-code" onclick="requestPairCode()" class="px-4 py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold rounded-xl transition shadow-xs flex items-center gap-1.5 whitespace-nowrap">
								<i class="fa-solid fa-paper-plane"></i> Minta Kode
							</button>
						</div>
						<p id="code-error" class="text-xs text-rose-600 font-semibold hidden mt-1"></p>
					</div>

					<!-- Kode Display Box -->
					<div id="code-result-box" class="hidden p-4 bg-emerald-50 border border-emerald-200 rounded-xl text-center space-y-2">
						<p class="text-xs font-bold text-emerald-800 uppercase tracking-wider">KODE PAIRING ANDA:</p>
						<div id="code-display" class="text-3xl font-black font-mono tracking-widest text-emerald-900 py-1 bg-white border border-emerald-300 rounded-lg">
							---- ----
						</div>
						<div class="text-[12px] text-slate-700 text-left space-y-1 pt-2">
							<p class="font-bold text-emerald-800">Langkah di WhatsApp HP:</p>
							<ol class="list-decimal list-inside space-y-0.5 text-slate-600">
								<li>Buka WhatsApp ➔ Titik Tiga / Setelan ➔ <strong>Perangkat Tertaut</strong></li>
								<li>Ketuk <strong>Tautkan Perangkat</strong></li>
								<li>Ketuk pilihan <strong>"Tautkan dengan nomor telepon saja"</strong> di bagian paling bawah layar kamera</li>
								<li>Masukkan 8 karakter kode di atas</li>
							</ol>
						</div>
					</div>
				</div>

				<div class="pt-2 border-t border-slate-100 flex items-center justify-between text-xs text-slate-400">
					<span id="conn-status-text">● Menunggu tautan perangkat...</span>
					<a href="/wa/reset" onclick="return confirm('Reset sesi WhatsApp?')" class="text-rose-600 hover:underline">Reset Sesi</a>
				</div>
			</div>

			<script>
				function switchTab(tab) {
					const tabQr = document.getElementById('tab-qr');
					const tabCode = document.getElementById('tab-code');
					const btnQr = document.getElementById('tab-qr-btn');
					const btnCode = document.getElementById('tab-code-btn');

					if (tab === 'qr') {
						tabQr.classList.remove('hidden');
						tabCode.classList.add('hidden');
						btnQr.className = 'flex-1 py-2 rounded-lg bg-white text-emerald-700 shadow-xs transition flex items-center justify-center gap-1.5';
						btnCode.className = 'flex-1 py-2 rounded-lg text-slate-600 hover:text-slate-800 transition flex items-center justify-center gap-1.5';
					} else {
						tabQr.classList.add('hidden');
						tabCode.classList.remove('hidden');
						btnCode.className = 'flex-1 py-2 rounded-lg bg-white text-emerald-700 shadow-xs transition flex items-center justify-center gap-1.5';
						btnQr.className = 'flex-1 py-2 rounded-lg text-slate-600 hover:text-slate-800 transition flex items-center justify-center gap-1.5';
					}
				}

				let isRequesting = false;
				async function requestPairCode() {
					if (isRequesting) return;
					const phone = document.getElementById('phone-input').value.trim();
					const errEl = document.getElementById('code-error');
					const resBox = document.getElementById('code-result-box');
					const codeDisp = document.getElementById('code-display');
					const btn = document.getElementById('btn-get-code');

					errEl.classList.add('hidden');
					errEl.innerText = '';
					if (!phone) {
						errEl.innerText = 'Silakan masukkan nomor telepon terlebih dahulu.';
						errEl.classList.remove('hidden');
						return;
					}

					isRequesting = true;
					btn.disabled = true;
					btn.innerHTML = '<i class="fa-solid fa-spinner fa-spin"></i> Meminta...';

					try {
						const res = await fetch('/wa/pair-phone', {
							method: 'POST',
							headers: {'Content-Type': 'application/json'},
							body: JSON.stringify({ phone: phone })
						});
						const data = await res.json();
						if (data.status === 'success' && data.code) {
							codeDisp.innerText = data.code;
							resBox.classList.remove('hidden');
						} else {
							errEl.innerText = data.message || 'Gagal membuat kode pairing.';
							errEl.classList.remove('hidden');
						}
					} catch (e) {
						errEl.innerText = 'Terjadi kesalahan jaringan: ' + e.message;
						errEl.classList.remove('hidden');
					} finally {
						isRequesting = false;
						btn.disabled = false;
						btn.innerHTML = '<i class="fa-solid fa-paper-plane"></i> Minta Kode';
					}
				}

				async function requestNewQR() {
					try {
						await fetch('/wa/reconnect?format=json');
						setTimeout(() => {
							location.reload();
						}, 1000);
					} catch (e) {
						location.reload();
					}
				}

				// Polling status secara realtime (tiap 2.5 detik) tanpa refresh seluruh halaman
				setInterval(async () => {
					try {
						const res = await fetch('/wa/status');
						const json = await res.json();
						if (json && json.data) {
							if (json.data.is_logged_in && json.data.status === 'CONNECTED') {
								location.reload(); // Langsung reload ke tampilan Sukses Hijau
							}
						}
					} catch (e) {}
				}, 2500);
			</script>
		</body>
		</html>
	`, qrElement)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// PairPhone: Menghasilkan 8-digit kode pairing WhatsApp via nomor telepon
func (h *WhatsAppHandler) PairPhone(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Nomor telepon wajib diisi (misal: 08123456789 atau 628123456789)",
		})
		return
	}

	code, err := h.bot.PairPhone(req.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"code":   code,
	})
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
