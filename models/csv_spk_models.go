package models

// DTO untuk Laporan SPK Per Cust dari Database CSV (imp_sampurna)
type CsvSpkPerCustItem struct {
	Site      string  `gorm:"column:site" json:"site"`
	TglMasuk  *string `gorm:"column:tgl_masuk" json:"tgl_masuk"`
	CloseSpk  *string `gorm:"column:close_spk" json:"close_spk"`
	CloseDate *string `gorm:"column:close_date" json:"close_date"`
	Nopol     string  `gorm:"column:nopol" json:"nopol"`
	Cabin    string  `gorm:"column:cabin" json:"cabin"`
	CustCode string  `gorm:"column:cust_code" json:"cust_code"`
	Kategori string  `gorm:"column:kategori" json:"kategori"`
	Spk      string  `gorm:"column:spk" json:"spk"`
	SoId     string  `gorm:"column:so_id" json:"so_id"`
	Servis   float64 `gorm:"column:servis" json:"servis"`
	Part     float64 `gorm:"column:part" json:"part"`
	Ban      float64 `gorm:"column:ban" json:"ban"`
	Total    float64 `gorm:"column:total" json:"total"`
	Catatan  *string `gorm:"column:catatan" json:"catatan"`
}

type CsvSpkFilterOptions struct {
	Areas      []string `json:"areas"`
	Customers  []string `json:"customers"`
	Purposes   []string `json:"purposes"`
	LastUpdate string   `json:"last_update"`
}

type CsvSpkDetailMetrics struct {
	SpkCount   int     `json:"spk_count"`
	Service    float64 `json:"service"`
	Part       float64 `json:"part"`
	Total      float64 `json:"total"`
}

// DTO untuk Detail Service & Sparepart per Nopol
type CsvSpkDetailItem struct {
	Sku          string  `gorm:"column:sku" json:"sku"`
	Product      string  `gorm:"column:product" json:"product"`
	Qty          float64 `gorm:"column:qty" json:"qty"`
	UnitPrice    float64 `gorm:"column:unit_price" json:"unit_price"`
	TotalPrice   float64 `gorm:"column:total_price" json:"total_price"`
	Spk          string  `gorm:"column:spk" json:"spk"`
	Category     string  `gorm:"column:category" json:"category"`
	Statue       string  `gorm:"column:statue" json:"statue"`
	TglMasuk     string  `gorm:"column:tgl_masuk" json:"tgl_masuk"`
	Selesai      string  `gorm:"column:selesai" json:"selesai"`
	RawCloseDate *string `gorm:"column:raw_close_date" json:"raw_close_date"`
	SpkStatus    string  `gorm:"column:spk_status" json:"spk_status"` // "CLOSE", "HOLD", "OPEN"
}

type CsvSpkNoteItem struct {
	Spk   string `gorm:"column:spk" json:"spk"`
	Notes string `gorm:"column:notes" json:"notes"`
}

type CsvSpkDetailResponse struct {
	Nopol            string              `json:"nopol"`
	TglAwal          string              `json:"tgl_awal"`
	TglAkhir         string              `json:"tgl_akhir"`
	TglMasuk         string              `json:"tgl_masuk"`
	TotalSpk         int                 `json:"total_spk"`
	TotalService     float64             `json:"total_service"`
	TotalSparepart   float64             `json:"total_sparepart"`
	TotalKeseluruhan float64             `json:"total_keseluruhan"`
	Closed           CsvSpkDetailMetrics `json:"closed"`
	Hold             CsvSpkDetailMetrics `json:"hold"`
	Open             CsvSpkDetailMetrics `json:"open"`
	Notes            []CsvSpkNoteItem    `json:"notes"`
	Services         []CsvSpkDetailItem  `json:"services"`
	Spareparts       []CsvSpkDetailItem  `json:"spareparts"`
}

// DTO untuk Summary Per Cust (Dual View)
type CsvSpkSummaryMetrics struct {
	SpkCount   int     `json:"spk_count"`
	TotalJasa  float64 `json:"total_jasa"`
	TotalPart  float64 `json:"total_part"`
	TotalBan   float64 `json:"total_ban"`
	GrandTotal float64 `json:"grand_total"`
}

type CsvCategorySummary struct {
	Category string               `json:"category"`
	Total    CsvSpkSummaryMetrics `json:"total"`
	Closed   CsvSpkSummaryMetrics `json:"closed"`
	Open     CsvSpkSummaryMetrics `json:"open"`
	Hold     CsvSpkSummaryMetrics `json:"hold"`
}

type CsvCustomerSummary struct {
	Customer   string                `json:"customer"`
	Categories []CsvCategorySummary  `json:"categories"`
	Total      CsvSpkSummaryMetrics  `json:"total"`
	Closed     CsvSpkSummaryMetrics  `json:"closed"`
	Open       CsvSpkSummaryMetrics  `json:"open"`
	Hold       CsvSpkSummaryMetrics  `json:"hold"`
}

type CsvGrandTotalSummary struct {
	Total  CsvSpkSummaryMetrics `json:"total"`
	Closed CsvSpkSummaryMetrics `json:"closed"`
	Open   CsvSpkSummaryMetrics `json:"open"`
	Hold   CsvSpkSummaryMetrics `json:"hold"`
}

type CsvSummaryPerCustResponse struct {
	TglAwal    string               `json:"tgl_awal"`
	TglAkhir   string               `json:"tgl_akhir"`
	Customers  []CsvCustomerSummary `json:"customers"`
	GrandTotal CsvGrandTotalSummary `json:"grand_total"`
}

// DTO untuk Resume Per Kategori
type CsvResumeMatrixCell struct {
	Amount   float64 `json:"amount"`
	SpkCount int     `json:"spk_count"`
}

type CsvResumeCustomerRow struct {
	Customer string                         `json:"customer"`
	Values   map[string]CsvResumeMatrixCell `json:"values"`
	Total    CsvResumeMatrixCell            `json:"total"`
}

type CsvResumeMatrixSection struct {
	Title    string                 `json:"title"`
	Rows     []CsvResumeCustomerRow `json:"rows"`
	TotalRow CsvResumeCustomerRow   `json:"total_row"`
}

type CsvResumePerKategoriResponse struct {
	TglAwal           string                                     `json:"tgl_awal"`
	TglAkhir          string                                     `json:"tgl_akhir"`
	Categories        []string                                   `json:"categories"`
	GrandTotalMatrix  map[string]map[string]CsvResumeMatrixCell  `json:"grand_total_matrix"`
	GrandTotalSummary map[string]CsvResumeMatrixCell             `json:"grand_total_summary"`
	TotalSpkSection   CsvResumeMatrixSection                     `json:"total_spk_section"`
	SpkCloseSection   CsvResumeMatrixSection                     `json:"spk_close_section"`
	SpkOpenSection    CsvResumeMatrixSection                     `json:"spk_open_section"`
	SpkHoldSection    CsvResumeMatrixSection                     `json:"spk_hold_section"`
}

// DTO untuk Detail Drilldown Resume Per Kategori
type CsvResumeKategoriDetailItem struct {
	Nopol    string  `gorm:"column:nopol" json:"nopol"`
	Spk      int     `gorm:"column:spk" json:"spk"`
	Kategori string  `gorm:"column:kategori" json:"kategori"`
	Service  float64 `gorm:"column:service" json:"service"`
	Part     float64 `gorm:"column:part" json:"part"`
	Ban      float64 `gorm:"column:ban" json:"ban"`
	Total    float64 `gorm:"column:total" json:"total"`
	Note     string  `gorm:"column:note" json:"note"`
}

type CsvResumeKategoriDetailResponse struct {
	Customer string                        `json:"customer"`
	Status   string                        `json:"status"` // "ALL", "CLOSE", "OPEN"
	Category string                        `json:"category"`
	TglAwal  string                        `json:"tgl_awal"`
	TglAkhir string                        `json:"tgl_akhir"`
	Items    []CsvResumeKategoriDetailItem `json:"items"`
}
