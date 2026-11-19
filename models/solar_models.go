package models

import "time"

// PemakaianSolar represents the record for daily fuel consumption per shift per unit
type PemakaianSolar struct {
	ID                uint       `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Unit              string     `gorm:"column:unit;size:50;not null;uniqueIndex:idx_unit_tgl_shift,priority:1" json:"unit"`
	Tanggal           time.Time  `gorm:"column:tanggal;type:date;not null;uniqueIndex:idx_unit_tgl_shift,priority:2" json:"tanggal"`
	Shift             string     `gorm:"column:shift;size:20;not null;uniqueIndex:idx_unit_tgl_shift,priority:3" json:"shift"` // "Pagi" or "Malam"
	AwalHm            float64    `gorm:"column:awal_hm;type:decimal(10,2);default:0" json:"awal_hm"`
	AkhirHm           float64    `gorm:"column:akhir_hm;type:decimal(10,2);default:0" json:"akhir_hm"`
	TotalJamHm        float64    `gorm:"column:total_jam_hm;type:decimal(10,2);default:0" json:"total_jam_hm"`
	ColokanAwalCm     float64    `gorm:"column:colokan_awal_cm;type:decimal(10,2);default:0" json:"colokan_awal_cm"`
	TotalPakaiCm      float64    `gorm:"column:total_pakai_cm;type:decimal(10,2);default:0" json:"total_pakai_cm"`
	PakaiLiter        float64    `gorm:"column:pakai_liter;type:decimal(10,2);default:0" json:"pakai_liter"`
	ColokanAkhirCm    float64    `gorm:"column:colokan_akhir_cm;type:decimal(10,2);default:0" json:"colokan_akhir_cm"`
	TotalIsiCm        float64    `gorm:"column:total_isi_cm;type:decimal(10,2);default:0" json:"total_isi_cm"`
	IsiLiter          float64    `gorm:"column:isi_liter;type:decimal(10,2);default:0" json:"isi_liter"`
	AkhirSetelahIsiCm float64    `gorm:"column:akhir_setelah_isi_cm;type:decimal(10,2);default:0" json:"akhir_setelah_isi_cm"`
	SisaTangkiLiter   float64    `gorm:"column:sisa_tangki_liter;type:decimal(10,2);default:0" json:"sisa_tangki_liter"`
	RasioLiterPerHm   float64    `gorm:"column:rasio_liter_per_hm;type:decimal(10,2);default:0" json:"rasio_liter_per_hm"`
	CreatedAt         *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (PemakaianSolar) TableName() string {
	return "pemakaian_solar"
}

// PemakaianSolarFilter holds request filter parameters
type PemakaianSolarFilter struct {
	TglAwal  string `form:"tglAwal"`
	TglAkhir string `form:"tglAkhir"`
	Unit     string `form:"unit"`
	Shift    string `form:"shift"`
	Page     int    `form:"page,default=1"`
	Limit    int    `form:"limit,default=50"`
}

// SolarSummary holds aggregate KPIs
type SolarSummary struct {
	TotalRows       int64   `json:"total_rows"`
	TotalPakaiLiter float64 `json:"total_pakai_liter"`
	TotalIsiLiter   float64 `json:"total_isi_liter"`
	TotalJamHm     float64 `json:"total_jam_hm"`
	AvgLiterPerHm   float64 `json:"avg_liter_per_hm"`
}

// SolarSyncSnapshot represents the state of solar synchronization
type SolarSyncSnapshot struct {
	IsRunning  bool       `json:"is_running"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Status     string     `json:"status"` // "idle", "running", "success", "error"
	Message    string     `json:"message"`
	Unit       string     `json:"unit"`
	Inserted   int        `json:"inserted"`
	Updated    int        `json:"updated"`
	Total      int        `json:"total"`
	Logs       []string   `json:"logs"`
}

// ==========================================
// DT_SOLAR (MODUL IMPORT DATA SOLAR)
// ==========================================

// DtSolar represents a granular row in the dt_solar table (6 rows per 1 row file)
type DtSolar struct {
	ID      int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Tgl     *time.Time `gorm:"column:tgl;type:date" json:"tgl"`
	Unit    string     `gorm:"column:unit;size:25;not null;default:'0'" json:"unit"`
	Jam     int        `gorm:"column:jam;not null;default:0" json:"jam"`
	Hm      float64    `gorm:"column:hm;type:double;not null;default:0" json:"hm"`
	Awal    float64    `gorm:"column:awal;type:double;not null;default:0" json:"awal"`
	Akhir   float64    `gorm:"column:akhir;type:double;not null;default:0" json:"akhir"`
	Isi     float64    `gorm:"column:isi;type:double;not null;default:0" json:"isi"`
	Satuan  string     `gorm:"column:satuan;size:20;not null;default:'liter'" json:"satuan"`
	Created *time.Time `gorm:"column:created" json:"created"`
}

func (DtSolar) TableName() string {
	return "dt_solar"
}

type DtSolarFilter struct {
	TglAwal  string `form:"tglAwal"`
	TglAkhir string `form:"tglAkhir"`
	Unit     string `form:"unit"`
	Jam      string `form:"jam"`
	Page     int    `form:"page,default=1"`
	Limit    int    `form:"limit,default=50"`
}

type DtSolarSummary struct {
	TotalRows int64   `json:"total_rows"`
	TotalIsi  float64 `json:"total_isi"`
	TotalHm   float64 `json:"total_hm"`
	TotalUnit int64   `json:"total_unit"`
}

// ==========================================
// LAPORAN SOLAR LAPORAN STRUCTS
// ==========================================

// 1. Laporan Detail Harian
type LaporanDetailHarianFilter struct {
	TglAwal  string `form:"tglAwal"`
	TglAkhir string `form:"tglAkhir"`
	Unit     string `form:"unit"`
	Search   string `form:"search"`
	Page     int    `form:"page,default=1"`
	Limit    int    `form:"limit,default=50"`
}

type LaporanDetailHarianItem struct {
	ID          int     `json:"id"`
	Tgl         string  `json:"tgl"`
	Unit        string  `json:"unit"`
	HmAwal      float64 `json:"hm_awal"`
	HmAkhir     float64 `json:"hm_akhir"`
	TotalHm     float64 `json:"total_hm"`
	PakaiCm     float64 `json:"pakai_cm"`
	Konsumsi    float64 `json:"konsumsi"`
	PakaiLiter  float64 `json:"pakai_liter"`
	IsiLiter    float64 `json:"isi_liter"`
	LtrPerHm    float64 `json:"ltr_per_hm"`
	NettoLiter  float64 `json:"netto_liter"`
	Created     string  `json:"created"`
}

type LaporanDetailHarianSummary struct {
	TotalRecords    int64   `json:"total_records"`
	TotalHm         float64 `json:"total_hm"`
	TotalPakaiCm    float64 `json:"total_pakai_cm"`
	TotalPakaiLiter float64 `json:"total_pakai_liter"`
	TotalIsiLiter   float64 `json:"total_isi_liter"`
	AvgLtrPerHm     float64 `json:"avg_ltr_per_hm"`
	NettoLiter      float64 `json:"netto_liter"`
	TotalUnit       int64   `json:"total_unit"`
}

// 2. Laporan Rekap Bulanan / Periode per Unit
type LaporanRekapUnitFilter struct {
	TglAwal  string `form:"tglAwal"`
	TglAkhir string `form:"tglAkhir"`
	Unit     string `form:"unit"`
	GroupBy  string `form:"groupBy,default=unit"` // "unit" or "month_unit"
}

type LaporanRekapUnitItem struct {
	Period          string  `json:"period"`
	Unit            string  `json:"unit"`
	HariKerja       int64   `json:"hari_kerja"`
	MinHmAwal       float64 `json:"min_hm_awal"`
	MaxHmAkhir      float64 `json:"max_hm_akhir"`
	TotalHm         float64 `json:"total_hm"`
	AvgHmPerHari    float64 `json:"avg_hm_per_hari"`
	TotalPakaiCm    float64 `json:"total_pakai_cm"`
	Konsumsi        float64 `json:"konsumsi"`
	TotalPakaiLiter float64 `json:"total_pakai_liter"`
	TotalIsiLiter   float64 `json:"total_isi_liter"`
	NettoLiter      float64 `json:"netto_liter"`
	AvgPakaiPerHari float64 `json:"avg_pakai_per_hari"`
	LtrPerHm        float64 `json:"ltr_per_hm"`
	EfisiensiStatus string  `json:"efisiensi_status"` // "Efisien", "Moderat", "Tinggi"
}

type LaporanRekapUnitSummary struct {
	TotalUnit       int64   `json:"total_unit"`
	TotalHariKerja  int64   `json:"total_hari_kerja"`
	TotalHm         float64 `json:"total_hm"`
	TotalPakaiLiter float64 `json:"total_pakai_liter"`
	TotalIsiLiter   float64 `json:"total_isi_liter"`
	AvgLtrPerHm     float64 `json:"avg_ltr_per_hm"`
	NettoLiter      float64 `json:"netto_liter"`
}

// 3. Laporan Rekonsiliasi & Anomali
type LaporanRekonsiliasiFilter struct {
	TglAwal          string `form:"tglAwal"`
	TglAkhir         string `form:"tglAkhir"`
	Unit             string `form:"unit"`
	KategoriAnomali  string `form:"kategoriAnomali"`
	OnlyAnomali      bool   `form:"onlyAnomali"`
	Page             int    `form:"page,default=1"`
	Limit            int    `form:"limit,default=50"`
}

type LaporanRekonsiliasiItem struct {
	ID              int     `json:"id"`
	Tgl             string  `json:"tgl"`
	Unit            string  `json:"unit"`
	HmAwal          float64 `json:"hm_awal"`
	HmAkhir         float64 `json:"hm_akhir"`
	TotalHm         float64 `json:"total_hm"`
	PakaiCm         float64 `json:"pakai_cm"`
	Konsumsi        float64 `json:"konsumsi"`
	PakaiLiter      float64 `json:"pakai_liter"`
	IsiLiter        float64 `json:"isi_liter"`
	LtrPerHm        float64 `json:"ltr_per_hm"`
	NettoLiter      float64 `json:"netto_liter"`
	StatusAudit     string  `json:"status_audit"`     // "NORMAL", "PERINGATAN", "ANOMALI"
	JenisAnomali    string  `json:"jenis_anomali"`    // Deskripsi jenis anomali
	KeteranganAudit string  `json:"keterangan_audit"` // Penjelasan audit
}

type LaporanRekonsiliasiSummary struct {
	TotalAuditRows         int64   `json:"total_audit_rows"`
	TotalAnomaliRows       int64   `json:"total_anomali_rows"`
	TotalPeringatanRows     int64   `json:"total_peringatan_rows"`
	TotalNormalRows        int64   `json:"total_normal_rows"`
	TotalPakaiAnomaliLiter float64 `json:"total_pakai_anomali_liter"`
	TotalSelisihNettoLiter float64 `json:"total_selisih_netto_liter"`
}
