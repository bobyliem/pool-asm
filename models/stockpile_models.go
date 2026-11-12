package models

import "time"

// ApiStockpile represents a row in the api_stockpile table (imp_sampurna DB)
type ApiStockpile struct {
	ID      uint       `gorm:"primaryKey;column:id" json:"id"`
	NoTiket string     `gorm:"column:no_tiket;size:25;not null;default:'0'" json:"no_tiket"`
	Reff    string     `gorm:"column:reff;size:25;not null;default:'0'" json:"reff"`
	Nopol   string     `gorm:"column:nopol;size:25;not null;default:'0'" json:"nopol"`
	Dk      string     `gorm:"column:dk;size:5;not null;default:'0'" json:"dk"`
	Prd     string     `gorm:"column:prd;size:25;not null;default:'0'" json:"prd"`
	Src     string     `gorm:"column:src;size:25;not null;default:'0'" json:"src"`
	Dest    string     `gorm:"column:dest;size:255;not null;default:'0'" json:"dest"`
	Wsrc    float64    `gorm:"column:wsrc;not null;default:0" json:"wsrc"`
	Wdest   float64    `gorm:"column:wdest;not null;default:0" json:"wdest"`
	Tgl     *time.Time `gorm:"column:tgl" json:"tgl"`
	Updated *time.Time `gorm:"column:updated" json:"updated"`
	Created *time.Time `gorm:"column:created" json:"created"`
}

func (ApiStockpile) TableName() string {
	return "api_stockpile"
}

// TransferOutFilter holds query parameters for Transfer Out report
type TransferOutFilter struct {
	TglAwal  string `form:"tglawal" json:"tglawal"`
	TglAkhir string `form:"tglakhir" json:"tglakhir"`
	Prd      string `form:"prd" json:"prd"`
	Nopol    string `form:"nopol" json:"nopol"`
	Dest     string `form:"dest" json:"dest"`
}

// TransferOutItem represents a single outgoing stockpile row
type TransferOutItem struct {
	ID      uint    `gorm:"column:id" json:"id"`
	NoTiket string  `gorm:"column:no_tiket" json:"no_tiket"`
	Reff    string  `gorm:"column:reff" json:"reff"`
	Nopol   string  `gorm:"column:nopol" json:"nopol"`
	Dk      string  `gorm:"column:dk" json:"dk"`
	Prd     string  `gorm:"column:prd" json:"prd"`
	Src     string  `gorm:"column:src" json:"src"`
	Dest    string  `gorm:"column:dest" json:"dest"`
	Wsrc    float64 `gorm:"column:wsrc" json:"wsrc"`
	Wdest   float64 `gorm:"column:wdest" json:"wdest"`
	Selisih float64 `gorm:"column:selisih" json:"selisih"`
	Tgl     string  `gorm:"column:tgl" json:"tgl"`
	Updated string  `gorm:"column:updated" json:"updated"`
}

// TransferOutProductSummary holds per-product aggregation
type TransferOutProductSummary struct {
	Prd        string  `json:"prd"`
	JmlRit     int     `json:"jml_rit"`
	TotalWsrc  float64 `json:"total_wsrc"`
	TotalWdest float64 `json:"total_wdest"`
}

// TransferOutDestSummary holds per-destination aggregation
type TransferOutDestSummary struct {
	Dest       string  `json:"dest"`
	JmlRit     int     `json:"jml_rit"`
	TotalWsrc  float64 `json:"total_wsrc"`
	TotalWdest float64 `json:"total_wdest"`
}

// TransferOutUnitSummary holds per-unit (nopol) aggregation
type TransferOutUnitSummary struct {
	Nopol      string  `json:"nopol"`
	JmlRit     int     `json:"jml_rit"`
	TotalWsrc  float64 `json:"total_wsrc"`
	TotalWdest float64 `json:"total_wdest"`
}

// TransferOutSummary holds overall KPI summary for the filtered period
type TransferOutSummary struct {
	TotalRit           int                         `json:"total_rit"`
	TotalUnit          int                         `json:"total_unit"`
	TotalDest          int                         `json:"total_dest"`
	TotalWsrc          float64                     `json:"total_wsrc"`
	TotalWdest         float64                     `json:"total_wdest"`
	TotalRitPlant      int                         `json:"total_rit_plant"`
	TotalWsrcWithPlant float64                     `json:"total_wsrc_with_plant"`
	SelisihPlant       float64                     `json:"selisih_plant"`
	WastePercent       float64                     `json:"waste_percent"`
	AvgWsrcPerRit      float64                     `json:"avg_wsrc_per_rit"`
	ByProduct          []TransferOutProductSummary `json:"by_product"`
	ByDest             []TransferOutDestSummary    `json:"by_dest"`
	ByUnit             []TransferOutUnitSummary    `json:"by_unit"`
}

// TransferOutFilterOptions holds distinct dropdown values for Transfer Out filters
type TransferOutFilterOptions struct {
	Products     []string `json:"products"`
	Nopols       []string `json:"nopols"`
	Destinations []string `json:"destinations"`
}

// TransferOutResponse is the API response structure for Transfer Out
type TransferOutResponse struct {
	TglAwal    string             `json:"tgl_awal"`
	TglAkhir   string             `json:"tgl_akhir"`
	LastUpdate string             `json:"last_update"`
	Summary    TransferOutSummary `json:"summary"`
	Items      []TransferOutItem  `json:"items"`
}
