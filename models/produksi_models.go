package models

// ResumePerHariItem mewakili data baris resume produksi harian dari ss_pok
type ResumePerHariItem struct {
	Tgl         string  `gorm:"column:tgl" json:"tgl"`
	JmlRit      int     `gorm:"column:jmlRit" json:"jml_rit"`
	JmlUnit     int     `gorm:"column:jmlUnit" json:"jml_unit"`
	RitPerUnit  float64 `gorm:"column:ritPerUnit" json:"rit_per_unit"`
	TtlMuat     float64 `gorm:"column:ttlMuat" json:"ttl_muat"`
	TtlSetor    float64 `gorm:"column:ttlSetor" json:"ttl_setor"`
	TotalWaste  float64 `gorm:"column:totalWaste" json:"total_waste"`
	RataPerRit  float64 `gorm:"column:rataPerRit" json:"rata_per_rit"`
	WastePerRit float64 `gorm:"column:wastePerRit" json:"waste_per_rit"`
	TargetTon   float64 `json:"target_ton"`   // Target harian: 365 Ton
	SelisihTon  float64 `json:"selisih_ton"`  // TtlSetor - TargetTon (negatif = kurang, positif = surplus)
	Pencapaian  float64 `json:"pencapaian"`   // (TtlSetor / TargetTon) * 100
}

// ResumePerHariSummary mewakili akumulasi data ringkasan periode
type ResumePerHariSummary struct {
	TotalHari         int     `json:"total_hari"`
	TotalRit          int     `json:"total_rit"`
	AvgRitPerHari     float64 `json:"avg_rit_per_hari"`
	TotalUnitUnique   int     `json:"total_unit_unique"`
	AvgRitPerUnit     float64 `json:"avg_rit_per_unit"`
	TotalMuat         float64 `json:"total_muat"`
	TotalSetor        float64 `json:"total_setor"`
	TotalWaste        float64 `json:"total_waste"`
	AvgSetorPerRit    float64 `json:"avg_setor_per_rit"`
	AvgWastePerRit    float64 `json:"avg_waste_per_rit"`
	WastePercentage   float64 `json:"waste_percentage"`
	DailyTarget       float64 `json:"daily_target"`        // 365 Ton/hari
	TotalTarget       float64 `json:"total_target"`        // TotalHari * 365
	TotalSelisih      float64 `json:"total_selisih"`       // TotalSetor - TotalTarget
	AvgSetorPerHari   float64 `json:"avg_setor_per_hari"`  // TotalSetor / TotalHari
	KekuranganPerHari float64 `json:"kekurangan_per_hari"` // DailyTarget - AvgSetorPerHari (positif = kurang X ton/hari)
	PencapaianPersen  float64 `json:"pencapaian_persen"`   // (TotalSetor / TotalTarget) * 100
}

// ResumePerHariResponse format JSON response untuk frontend
type ResumePerHariResponse struct {
	TglAwal    string               `json:"tgl_awal"`
	TglAkhir   string               `json:"tgl_akhir"`
	LastUpdate string               `json:"last_update"`
	Summary    ResumePerHariSummary `json:"summary"`
	Items      []ResumePerHariItem  `json:"items"`
}

// SummaryPerDriverItem mewakili data baris summary produksi per sopir dari ss_pok
type SummaryPerDriverItem struct {
	Sopir       string  `gorm:"column:sopir" json:"sopir"`
	JmlRit      int     `gorm:"column:jmlRit" json:"jml_rit"`
	JmlHari     int     `gorm:"column:jmlHari" json:"jml_hari"`
	RitPerHari  float64 `gorm:"column:ritPerHari" json:"rit_per_hari"`
	TtlMuat     float64 `gorm:"column:ttlMuat" json:"ttl_muat"`
	TtlSetor    float64 `gorm:"column:ttlSetor" json:"ttl_setor"`
	TotalWaste  float64 `gorm:"column:totalWaste" json:"total_waste"`
	RataPerRit  float64 `gorm:"column:rataPerRit" json:"rata_per_rit"`
	WastePerRit float64 `gorm:"column:wastePerRit" json:"waste_per_rit"`
}

// SummaryPerDriverSummary mewakili akumulasi data ringkasan per sopir
type SummaryPerDriverSummary struct {
	TotalSopir      int     `json:"total_sopir"`
	TotalRit        int     `json:"total_rit"`
	AvgRitPerDriver float64 `json:"avg_rit_per_driver"`
	TotalMuat       float64 `json:"total_muat"`
	TotalSetor      float64 `json:"total_setor"`
	TotalWaste      float64 `json:"total_waste"`
	AvgSetorPerRit  float64 `json:"avg_setor_per_rit"`
	AvgWastePerRit  float64 `json:"avg_waste_per_rit"`
	WastePercentage float64 `json:"waste_percentage"`
}

// SummaryPerDriverResponse format JSON response untuk frontend
type SummaryPerDriverResponse struct {
	TglAwal    string                  `json:"tgl_awal"`
	TglAkhir   string                  `json:"tgl_akhir"`
	LastUpdate string                  `json:"last_update"`
	Summary    SummaryPerDriverSummary `json:"summary"`
	Items      []SummaryPerDriverItem  `json:"items"`
}

// MonitoringRitaseCell data ritase untuk 1 unit pada 1 tanggal tertentu
type MonitoringRitaseCell struct {
	AmbilDo  int     `json:"ambil_do"`
	VoidRit  int     `json:"void_rit"`
	SjRit    int     `json:"sj_rit"`
	VolAsal  float64 `json:"vol_asal"`
	VolAkhir float64 `json:"vol_akhir"`
	Selisih  float64 `json:"selisih"`
}

// MonitoringRitaseRow baris tabel per unit (nopol & supir) beserta data per tanggal
type MonitoringRitaseRow struct {
	Nopol string                          `json:"nopol"`
	Sopir string                          `json:"sopir"`
	Data  map[string]MonitoringRitaseCell `json:"data"` // key: YYYY-MM-DD
}

// MonitoringRitaseTotal total kolom per tanggal
type MonitoringRitaseTotal struct {
	AmbilDo  int     `json:"ambil_do"`
	VoidRit  int     `json:"void_rit"`
	SjRit    int     `json:"sj_rit"`
	VolAsal  float64 `json:"vol_asal"`
	VolAkhir float64 `json:"vol_akhir"`
	Selisih  float64 `json:"selisih"`
}

// MonitoringRitaseResponse payload response untuk endpoint Monitoring Ritase
type MonitoringRitaseResponse struct {
	TglAwal    string                           `json:"tgl_awal"`
	TglAkhir   string                           `json:"tgl_akhir"`
	LastUpdate string                           `json:"last_update"`
	Dates      []string                         `json:"dates"`      // list tanggal berurutan YYYY-MM-DD
	Rows       []MonitoringRitaseRow            `json:"rows"`       // 13 nopol
	Totals     map[string]MonitoringRitaseTotal `json:"totals"`     // key: YYYY-MM-DD
	GrandTotal MonitoringRitaseTotal            `json:"grand_total"` // akumulasi seluruh tanggal & unit
}
