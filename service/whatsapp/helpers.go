package whatsapp

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// FormatRupiah memformat angka ke dalam format mata uang Rupiah (contoh: Rp 1.500.000)
func FormatRupiah(amount float64) string {
	negative := false
	if amount < 0 {
		negative = true
		amount = -amount
	}

	rounded := int64(amount + 0.5)
	str := fmt.Sprintf("%d", rounded)

	n := len(str)
	if n <= 3 {
		if negative {
			return "-Rp " + str
		}
		return "Rp " + str
	}

	var parts []string
	remainder := n % 3
	if remainder > 0 {
		parts = append(parts, str[:remainder])
	}
	for i := remainder; i < n; i += 3 {
		parts = append(parts, str[i:i+3])
	}

	res := "Rp " + strings.Join(parts, ".")
	if negative {
		return "-" + res
	}
	return res
}

// CleanPhone membersihkan nomor HP dari format WhatsApp (@s.whatsapp.net, @lid, dll)
// mengembalikan raw (hanya digit), phone0 (08xx), dan phone62 (628xx)
func CleanPhone(phone string) (raw string, phone0 string, phone62 string) {
	parts := strings.Split(phone, "@")
	re := regexp.MustCompile(`\D+`)
	raw = re.ReplaceAllString(parts[0], "")

	if strings.HasPrefix(raw, "62") {
		phone0 = "0" + raw[2:]
		phone62 = raw
	} else if strings.HasPrefix(raw, "0") {
		phone0 = raw
		phone62 = "62" + raw[1:]
	} else if raw != "" {
		phone0 = "0" + raw
		phone62 = "62" + raw
	}

	return raw, phone0, phone62
}

// IsAdmin mengecek apakah nomor HP terdaftar sebagai Admin di tabel settings
func IsAdmin(db *gorm.DB, phone string) bool {
	if db == nil {
		return false
	}
	_, p0, p62 := CleanPhone(phone)

	var count int64
	err := db.Table("settings").
		Where("no_hp = ? OR no_hp = ?", p0, p62).
		Count(&count).Error

	return err == nil && count > 0
}

// CheckUserAccess mengecek otorisasi akses (admin atau user statue=9)
func CheckUserAccess(db *gorm.DB, phone string) (bool, string) {
	if db == nil {
		return false, "Database tidak terhubung."
	}

	// 1. Cek Admin
	if IsAdmin(db, phone) {
		return true, ""
	}

	// 2. Cek User di tabel users
	_, p0, p62 := CleanPhone(phone)
	var user struct {
		ID     int64 `gorm:"column:id"`
		Statue int   `gorm:"column:statue"`
	}

	err := db.Table("users").
		Select("id, statue").
		Where("no_hp = ? OR no_hp = ?", p0, p62).
		First(&user).Error

	if err != nil {
		return false, "⛔ Nomor Anda belum terdaftar.\nSilakan lakukan registrasi terlebih dahulu dengan mengetik */registrasi*."
	}

	switch user.Statue {
	case 9:
		return true, ""
	case 0:
		return false, "Anda sudah masuk dalam list antrian kami, silahkan follow up ke admin kami."
	case 4:
		return false, "Status akun Anda sudah inactive, mohon hubungi admin kami untuk konfirmasi lebih lanjut."
	default:
		return false, fmt.Sprintf("Status akun Anda: %d. Silahkan hubungi admin kami.", user.Statue)
	}
}
