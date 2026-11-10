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
