package whatsapp_test

import (
	"fmt"
	"testing"
	"time"

	"asm/service/whatsapp"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const sampleWA = `Laporan solar
Tgl:24/11/26
Unit:sk05
Hm,:20852,7
Colokan 33cm
Isi solar:(jerigen)
Sesudah:(cm)

Laporan solar
Tgl:23/11/26
Unit: komatsu 
Hm:01372,5
Colokan 42cm
Isi solar:(jerigen)
Sesudah(cm)

Laporan solar 
Tgl 24 /11/ 20026
SK 200 (04 bleker
Clkn sebelum di isi 19 cm
Di isi 10 jerigen clkn 59
7427,9

Laporan Solar
Tanggal: 24nov.2026
Unit: SK.330.02
HM: 46143,5
Colokan..35.cm.star pagi
Isi Solar (cm)
Sesudah (cm)

Laporan Solar
Tanggal: .24.nov.2026
Unit: 330.01
HM: 14935,3
Colokan 56.cm ga isisolar.star.pagi
Isi Solar (Jerigen):
Sesudah (CM):

Laporan Solar
Tanggal: 24.nov.2026
Unit: SK.200.03
HM: 30644,0
Colokan.pagi.75.cm
Isi Solar (cm)
Sesudah (cm)

Laporan solar
Tgl:24/11/26
Unit:sk05
Hm20857,0h
Colokan awal 17cm
Isi solar:10jerigen colokan 71cm

----- 
Laporan solar
Tgl 24 NOV 2026
SK 200 (04 bleker
Cokan solar 39 cm
hm: 7434,9

Laporan solar 
Tgl:24/11/26
Unit komatsu
Hm:01376.3h
Colokan 23cm
Isi solar(jerigen)
Sesudah(cm)

Laporan solar
Tgl:24/11/26
Unit:sk05
Hm:20860.2h
Colokan:53cm
Isi solar(jerigen)
Sesudah(cm)

Laporan Solar
Tanggal: 24.nov2026
Unit: LDR.cat.950.g.02
HM: 20166,5
Colokan.80cm.ga isisolar. loder brigdon
Isi Solar (Jerigen):
Sesudah (CM):

Laporan Solar
Tanggal: 24nov.2026
Unit: SK.330.02
HM: 46147,6
Colokan..22.cm.ga isi solar
Isi Solar (cm)
Sesudah (cm)

Laporan solar
01/330/43/cm
HM:14942, 0

Laporan Solar
Tanggal: 24.nov.2026
Unit: SK.200.03
HM: 30649,4
Colokan.58.cm.ga isi
Isi Solar (cm)
Sesudah (cm)

Laporan solar
Tgl:24/11/26
Unit:sk05
Hm:20861,5h
Colokan, 50cm
Isi solar:(jerigen)
Sesudah:(cm)

Laporan Solar
Tanggal: .24.nov.2026.malam
Unit: 330.01
HM: 14943,1
Colokan 40.ga isi
Isi Solar (Jerigen):
Sesudah (CM):`

func TestParseSolarWA(t *testing.T) {
	dsn := "root:P@ssw0rd@tcp(127.0.0.1:3307)/imp_sampurna?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect DB: %v", err)
	}

	now := time.Now()
	result := whatsapp.ParseSolarWA(db, sampleWA, &now)

	t.Logf("Total reports parsed: %d", len(result.Reports))
	t.Logf("Total errors: %d", len(result.Errors))

	for i, r := range result.Reports {
		t.Logf("[%2d] Tgl=%s | Jenis=%-7s | Unit=%-15s | RawUnit=%-20s | HM=%.1f | IAwal=%-7s (%.1f L) | IIsi=%-11s (%.1f L) | IAkhir=%-7s (%.1f L)",
			i+1, r.Tanggal.Format("2006-01-02"), r.JenisDt, r.Unit, r.RawUnit, r.Hm,
			r.IAwal, r.Awal, r.IIsi, r.Isi, r.IAkhir, r.Akhir)
	}

	for _, errStr := range result.Errors {
		t.Errorf("Error during parse: %s", errStr)
	}
}


func TestSaveParsedReports(t *testing.T) {
	dsn := "root:P@ssw0rd@tcp(127.0.0.1:3307)/imp_sampurna?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect DB: %v", err)
	}

	now := time.Now()
	result := whatsapp.ParseSolarWA(db, sampleWA, &now)
	if len(result.Reports) == 0 {
		t.Fatalf("No reports parsed")
	}

	// Clean table first for testing
	db.Exec("TRUNCATE TABLE dt_solar_autowa")

	inserted, errors := whatsapp.SaveParsedReports(db, result.Reports)
	t.Logf("Inserted: %d rows", inserted)
	for _, e := range errors {
		t.Errorf("Insert error: %s", e)
	}

	// Verify from database
	var count int64
	db.Table("dt_solar_autowa").Count(&count)
	t.Logf("Rows in dt_solar_autowa after test: %d", count)
	if count != int64(len(result.Reports)) {
		t.Errorf("Expected %d rows, got %d", len(result.Reports), count)
	}

	// Print bot reply text simulation
	reply := whatsapp.FormatParseResult(result, inserted, errors)
	t.Logf("\n=== SIMULATED BOT WHATSAPP REPLY ===\n%s\n", reply)
}


func TestGroupFilter(t *testing.T) {
	dsn := "root:P@ssw0rd@tcp(127.0.0.1:3307)/imp_sampurna?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect DB: %v", err)
	}

	// 1. Set target group
	targetJID := "120363028372618291@g.us"
	db.Exec("INSERT INTO wa_config (conf_key, conf_value) VALUES ('wa_solar_group_id', ?) ON DUPLICATE KEY UPDATE conf_value = VALUES(conf_value)", targetJID)

	got := whatsapp.GetTargetSolarGroup(db)
	if got != targetJID {
		t.Errorf("Expected target group %s, got %s", targetJID, got)
	}
	t.Logf("Configured target solar group: %s", got)

	// Clean up after test
	db.Exec("DELETE FROM wa_config WHERE conf_key IN ('wa_solar_group_id', 'wa_solar_group_name')")
	cleared := whatsapp.GetTargetSolarGroup(db)
	if cleared != "" {
		t.Errorf("Expected empty after cleanup, got %s", cleared)
	}
	t.Logf("Cleaned up successfully, current target group: '%s'", cleared)
}


func TestParseDateWithTypo(t *testing.T) {
	dsn := "root:P@ssw0rd@tcp(127.0.0.1:3307)/imp_sampurna?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect DB: %v", err)
	}

	testWA := `Laporan solar
Tgl: 26 NOV 026
Unit: sk05
Hm: 20865,0
Colokan 35cm
Isi solar: (jerigen)
Sesudah: (cm)`

	now := time.Now()
	res := whatsapp.ParseSolarWA(db, testWA, &now)
	if len(res.Reports) != 1 {
		t.Fatalf("Expected 1 report, got %d, errors: %v", len(res.Reports), res.Errors)
	}

	r := res.Reports[0]
	expectedDate := "2026-11-26"
	if r.Tanggal.Format("2006-01-02") != expectedDate {
		t.Errorf("Expected date %s, got %s", expectedDate, r.Tanggal.Format("2006-01-02"))
	}
	t.Logf("Successfully parsed date with typo '26 NOV 026' -> %s", r.Tanggal.Format("2006-01-02"))
}


func TestAlphanumericUnitMatching(t *testing.T) {
	dsn := "root:P@ssw0rd@tcp(127.0.0.1:3307)/imp_sampurna?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect DB: %v", err)
	}

	testCases := []struct {
		inputUnit    string
		expectedUnit string
	}{
		{"sk05", "SK200-10 05"},
		{"SK.05", "SK200-10 05"},
		{"SK-05", "SK200-10 05"},
		{"SK 05", "SK200-10 05"},
		{"komatsu", "PC200-8"},
		{"KOMATSU.", "PC200-8"},
		{"SK 200 (04 bleker", "KOB 04"},
		{"SK200-04", "KOB 04"},
		{"SK.330.02", "SK330-8 02"},
		{"sk 330 02", "SK330-8 02"},
		{"sk.330.02", "SK330-8 02"},
		{"330.01", "SK330-8"},
		{"330 01", "SK330-8"},
		{"330-01", "SK330-8"},
		{"LDR.cat.950.g.02", "CAT 950G 02"},
		{"ldr cat 950 g 02", "CAT 950G 02"},
		{"SK.200.03", "KOB 03"},
		{"KOB 03", "KOB 03"},
		{"kob-03", "KOB 03"},
	}

	now := time.Now()
	for _, tc := range testCases {
		msg := fmt.Sprintf("Laporan solar\nTgl: 27/11/2026\nUnit: %s\nHm: 1000\nColokan 30cm\nIsi solar: (jerigen)\nSesudah: (cm)", tc.inputUnit)
		res := whatsapp.ParseSolarWA(db, msg, &now)
		if len(res.Reports) == 0 {
			t.Errorf("Failed to parse unit '%s': %v", tc.inputUnit, res.Errors)
			continue
		}
		got := res.Reports[0].Unit
		if got != tc.expectedUnit {
			t.Errorf("Unit '%s' matched to '%s', expected '%s'", tc.inputUnit, got, tc.expectedUnit)
		} else {
			t.Logf("✓ '%s' successfully matched to '%s'", tc.inputUnit, got)
		}
	}
}
