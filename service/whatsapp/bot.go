package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"asm/service"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type WhatsAppBot struct {
	client      *whatsmeow.Client
	container   *sqlstore.Container
	dbCsv       *gorm.DB
	syncService service.CsvSyncService

	qrText     string
	qrPNG      []byte
	isLoggedIn bool
	isConnected bool

	dedupMap sync.Map
	mu       sync.RWMutex
}

func NewWhatsAppBot(dbCsv *gorm.DB, syncService service.CsvSyncService) *WhatsAppBot {
	return &WhatsAppBot{
		dbCsv:       dbCsv,
		syncService: syncService,
	}
}

// Start menginisialisasi client WhatsMeow dan mulai koneksi ke WhatsApp
func (b *WhatsAppBot) Start() error {
	// 1. Siapkan folder data untuk file database session sqlite
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "whatsapp_session.db")
	dbUri := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", dbPath)

	logLevel := "INFO"
	clientLog := waLog.Stdout("WhatsMeow", logLevel, true)

	container, err := sqlstore.New(context.Background(), "sqlite", dbUri, clientLog)
	if err != nil {
		return fmt.Errorf("failed to init sqlstore: %w", err)
	}
	b.container = container

	// 2. Ambil atau buat device store
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get first device: %w", err)
	}

	b.client = whatsmeow.NewClient(deviceStore, clientLog)
	b.client.AddEventHandler(b.eventHandler)

	// 3. Cek apakah sudah login sebelumnya
	if b.client.Store.ID == nil {
		// Device baru: butuh Scan QR
		qrChan, err := b.client.GetQRChannel(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get qr channel: %w", err)
		}

		err = b.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect client: %w", err)
		}

		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					b.mu.Lock()
					b.qrText = evt.Code
					// Generate PNG untuk akses via Web Browser (http://localhost:8080/wa/qr)
					png, err := qrcode.Encode(evt.Code, qrcode.Medium, 300)
					if err == nil {
						b.qrPNG = png
					}
					b.mu.Unlock()

					// Tampilkan QR Code langsung di Terminal Console
					fmt.Println("\n==================================================================")
					fmt.Println("📲 SCAN QR CODE INI DENGAN WHATSAPP (Perangkat Tertaut):")
					fmt.Println("   Atau buka di browser: http://localhost:8080/wa/qr")
					fmt.Println("==================================================================")
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					fmt.Println("==================================================================")
				} else if evt.Event == "success" {
					b.mu.Lock()
					b.isLoggedIn = true
					b.qrText = ""
					b.qrPNG = nil
					b.mu.Unlock()
					fmt.Println("✅ [WhatsApp] Login WhatsApp BERHASIL! Session tersimpan.")
				}
			}
		}()
	} else {
		// Sudah ada session tersimpan
		b.mu.Lock()
		b.isLoggedIn = true
		b.mu.Unlock()

		err = b.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to reconnect existing session: %w", err)
		}
		fmt.Printf("✅ [WhatsApp] Terkoneksi kembali dengan nomor: %s\n", b.client.Store.ID.User)
	}

	// Rutin pembersihan deduplikasi setiap 5 menit
	go b.cleanupDedupLoop()

	return nil
}

// Stop mematikan koneksi WhatsApp dengan aman
func (b *WhatsAppBot) Stop() {
	if b.client != nil {
		b.client.Disconnect()
	}
	if b.container != nil {
		b.container.Close()
	}
}

// ReconnectQR mencoba rekoneksi untuk mendapatkan channel QR baru jika belum login dan offline
func (b *WhatsAppBot) ReconnectQR() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isLoggedIn || b.client == nil || b.client.IsConnected() {
		return nil
	}

	qrChan, err := b.client.GetQRChannel(context.Background())
	if err != nil {
		return err
	}

	if err := b.client.Connect(); err != nil {
		return err
	}

	go func() {
		for evt := range qrChan {
			if evt.Event == "code" {
				b.mu.Lock()
				b.qrText = evt.Code
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 300)
				if err == nil {
					b.qrPNG = png
				}
				b.mu.Unlock()

				fmt.Println("\n==================================================================")
				fmt.Println("📲 SCAN QR CODE INI DENGAN WHATSAPP (Perangkat Tertaut):")
				fmt.Println("   Atau buka di browser: http://localhost:8080/wa/qr")
				fmt.Println("==================================================================")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("==================================================================")
			} else if evt.Event == "success" {
				b.mu.Lock()
				b.isLoggedIn = true
				b.qrText = ""
				b.qrPNG = nil
				b.mu.Unlock()
				fmt.Println("✅ [WhatsApp] Login WhatsApp BERHASIL! Session tersimpan.")
			}
		}
	}()

	return nil
}

// GetQR mengembalikan teks QR, PNG bytes, dan status login
func (b *WhatsAppBot) GetQR() (string, []byte, bool) {
	b.mu.RLock()
	qrText := b.qrText
	qrPNG := b.qrPNG
	isLoggedIn := b.isLoggedIn
	isConnected := b.client != nil && b.client.IsConnected()
	b.mu.RUnlock()

	if !isLoggedIn && !isConnected {
		go func() {
			_ = b.ReconnectQR()
		}()
	}

	return qrText, qrPNG, isLoggedIn
}

// GetStatus mengembalikan status detail koneksi
func (b *WhatsAppBot) GetStatus() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	status := "OFFLINE"
	phone := "-"
	if b.client != nil {
		if b.client.IsConnected() {
			status = "CONNECTED"
		}
		if b.client.Store != nil && b.client.Store.ID != nil {
			phone = b.client.Store.ID.User
		}
	}

	return map[string]interface{}{
		"status":       status,
		"is_logged_in": b.isLoggedIn,
		"phone":        phone,
		"time":         time.Now().Format("2006-01-02 15:04:05"),
	}
}

// SendMessage mengirim pesan teks ke nomor WhatsApp penerima
func (b *WhatsAppBot) SendMessage(recipientPhone, text string) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("client whatsapp not connected")
	}

	_, _, p62 := CleanPhone(recipientPhone)
	if p62 == "" {
		return fmt.Errorf("invalid recipient phone: %s", recipientPhone)
	}

	recipientJID := types.NewJID(p62, types.DefaultUserServer)
	msg := &waProto.Message{
		Conversation: proto.String(text),
	}

	_, err := b.client.SendMessage(context.Background(), recipientJID, msg)
	return err
}

// eventHandler menangani semua event pesan masuk dari WhatsApp
func (b *WhatsAppBot) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		b.mu.Lock()
		b.isConnected = true
		b.mu.Unlock()

	case *events.LoggedOut:
		b.mu.Lock()
		b.isLoggedIn = false
		b.isConnected = false
		b.mu.Unlock()
		fmt.Println("⚠️ [WhatsApp] Bot telah Logged Out dari WhatsApp.")

	case *events.Message:
		// 1. Abaikan pesan dari diri sendiri
		if v.Info.IsFromMe {
			return
		}

		// 2. Abaikan status/broadcast
		if v.Info.Chat.Server == types.BroadcastServer || v.Info.Chat.User == "status" {
			return
		}

		// 3. Anti-Duplikat: Cek Message ID
		msgID := v.Info.ID
		if _, exists := b.dedupMap.LoadOrStore(msgID, time.Now()); exists {
			return // Pesan duplikat diabaikan
		}

		// 4. Ekstrak isi teks pesan
		body := ""
		if v.Message.GetConversation() != "" {
			body = v.Message.GetConversation()
		} else if v.Message.GetExtendedTextMessage() != nil {
			body = v.Message.GetExtendedTextMessage().GetText()
		}

		body = strings.TrimSpace(body)
		if body == "" {
			return
		}

		// 5. Resolusi Nomor Pengirim untuk Cek Hak Akses (konversi LID ke nomor telepon asli jika perlu)
		senderPhone := b.resolveSenderPhone(v)
		fmt.Printf("[WhatsApp] Pesan dari %s (Alt: %s, Chat: %s) -> Nomor Asli: %s | Pesan: %s\n",
			v.Info.Sender, v.Info.SenderAlt, v.Info.Chat, senderPhone, body)

		// Jika dari grup, senderJID adalah orangnya. Chat target reply adalah grupnya.
		targetChat := v.Info.Chat

		// 6. Jalankan router perintah
		b.handleIncomingCommand(targetChat, v.Info.Sender, senderPhone, body)
	}
}

// resolveSenderPhone menyelesaikan nomor telepon pengirim dari berbagai format WhatsApp (termasuk Privacy LID)
func (b *WhatsAppBot) resolveSenderPhone(msg *events.Message) string {
	info := msg.Info

	// 1. Jika SenderAlt terisi nomor telepon (WhatsApp mengisi participant_pn / sender_pn saat Sender adalah LID)
	if (info.SenderAlt.Server == types.DefaultUserServer || info.SenderAlt.Server == types.LegacyUserServer) && info.SenderAlt.User != "" {
		return info.SenderAlt.User
	}

	// 2. Jika Sender sendiri adalah nomor telepon biasa (@s.whatsapp.net)
	if (info.Sender.Server == types.DefaultUserServer || info.Sender.Server == types.LegacyUserServer) && info.Sender.User != "" {
		return info.Sender.User
	}

	// 3. Untuk DM langsung (bukan grup), cek apakah Chat JID adalah nomor telepon
	if !info.IsGroup && (info.Chat.Server == types.DefaultUserServer || info.Chat.Server == types.LegacyUserServer) && info.Chat.User != "" {
		return info.Chat.User
	}

	// 4. Cari dari mapping internal whatsmeow LID Store (whatsmeow_lid_map di SQLite)
	if b.client != nil && b.client.Store != nil && b.client.Store.LIDs != nil {
		if pnJID, err := b.client.Store.LIDs.GetPNForLID(context.Background(), info.Sender); err == nil && !pnJID.IsEmpty() {
			if pnJID.User != "" {
				return pnJID.User
			}
		}
		if !info.IsGroup {
			if pnJID, err := b.client.Store.LIDs.GetPNForLID(context.Background(), info.Chat); err == nil && !pnJID.IsEmpty() {
				if pnJID.User != "" {
					return pnJID.User
				}
			}
		}
	}

	// Fallback jika tidak ditemukan
	return info.Sender.User
}

// handleIncomingCommand memproses keyword dan mengirimkan balasan
func (b *WhatsAppBot) handleIncomingCommand(chatJID, senderJID types.JID, senderPhone, body string) {
	parts := strings.Fields(body)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])
	parameter := ""
	if len(parts) > 1 {
		// Ambil sisa teks setelah keyword
		idx := strings.Index(body, parts[0]) + len(parts[0])
		parameter = strings.TrimSpace(body[idx:])
	}

	// Daftar Perintah yang diproteksi otorisasi
	protectedCommands := map[string]bool{
		"/spk": true, "/nopol": true, "/omsetclosing": true,
		"/omset": true, "/omsetpending": true, "/infolelang": true,
		"/updatelelang": true, "/tarikdata": true, "/sync": true,
		"/help": true, "/bantuan": true, "/helpadmin": true, "/bantuanadmin": true,
	}

	if protectedCommands[command] {
		allowed, accessMsg := CheckUserAccess(b.dbCsv, senderPhone)
		if !allowed {
			b.replyText(chatJID, accessMsg)
			return
		}
	}

	var reply string

	switch command {
	case "/spk":
		reply = HandleCommandSpk(b.dbCsv, parameter)

	case "/nopol":
		reply = HandleCommandNopol(b.dbCsv, parameter)

	case "/omsetclosing":
		// Auto trigger sync jika perlu
		b.autoTriggerSync()
		reply = HandleCommandOmsetClosing(b.dbCsv)

	case "/omset":
		b.autoTriggerSync()
		reply = HandleCommandOmset(b.dbCsv, parameter)

	case "/omsetpending":
		b.autoTriggerSync()
		reply = HandleCommandOmsetPending(b.dbCsv, parameter)

	case "/infolelang":
		reply = HandleCommandInfoLelang(b.dbCsv, parameter)

	case "/updatelelang":
		reply = HandleCommandUpdateLelang(b.dbCsv, parameter)

	case "/registrasi":
		reply = HandleCommandRegistrasi(b.dbCsv, b, senderJID, senderPhone, parameter)

	case "/aktifasi":
		reply = HandleCommandAktifasi(b.dbCsv, b, senderPhone, parameter)

	case "/tarikdata", "/sync":
		reply = b.handleCommandTarikData(senderPhone)

	case "/help", "/bantuan":
		reply = HandleCommandHelp()

	case "/helpadmin", "/bantuanadmin":
		reply = HandleCommandHelpAdmin(b.dbCsv, senderPhone)

	default:
		// Bukan keyword terdaftar, abaikan
		return
	}

	if reply != "" {
		b.replyText(chatJID, reply)
	}
}

// replyText mengirim balasan langsung ke chat pengirim
func (b *WhatsAppBot) replyText(chatJID types.JID, text string) {
	if b.client == nil || !b.client.IsConnected() {
		return
	}

	msg := &waProto.Message{
		Conversation: proto.String(text),
	}

	_, err := b.client.SendMessage(context.Background(), chatJID, msg)
	if err != nil {
		fmt.Printf("⚠️ [WhatsApp] Error sending reply to %s: %v\n", chatJID.String(), err)
	}
}

// handleCommandTarikData: sinkronisasi spreadsheet manual oleh admin
func (b *WhatsAppBot) handleCommandTarikData(senderPhone string) string {
	if !IsAdmin(b.dbCsv, senderPhone) {
		return "⛔ Maaf, perintah */tarikdata* hanya dapat dijalankan oleh Admin."
	}

	if b.syncService == nil {
		return "❌ Service sinkronisasi data tidak tersedia."
	}

	success, msg := b.syncService.TriggerSync("")
	if !success {
		return fmt.Sprintf("⏳ *SINKRONISASI DATA SEDANG BERJALAN*\n------------------------------\n%s", msg)
	}

	return fmt.Sprintf("✅ *SINKRONISASI DATA BERHASIL*\n------------------------------\nData CSV telah selesai ditarik dan diperbarui ke database.\n\nSilakan ketik */omset* untuk melihat rekap data terbaru.")
}

// autoTriggerSync menjalankan sinkronisasi jika data terakhir sudah > 3 jam
func (b *WhatsAppBot) autoTriggerSync() {
	if b.dbCsv == nil || b.syncService == nil {
		return
	}

	var check struct {
		DiffSec int64 `gorm:"column:diff_sec"`
	}
	err := b.dbCsv.Raw(`
		SELECT TIMESTAMPDIFF(SECOND, COALESCE(finished, started, created), NOW()) AS diff_sec
		FROM log_imp
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&check).Error

	if err == nil && check.DiffSec > 10800 {
		go b.syncService.TriggerSync("")
	}
}

// cleanupDedupLoop membersihkan map anti-duplikat tiap 5 menit
func (b *WhatsAppBot) cleanupDedupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		b.dedupMap.Range(func(key, value interface{}) bool {
			if t, ok := value.(time.Time); ok {
				if now.Sub(t) > 2*time.Minute {
					b.dedupMap.Delete(key)
				}
			}
			return true
		})
	}
}
