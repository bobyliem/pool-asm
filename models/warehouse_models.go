package models

import (
	"fmt"
	"time"
)

// CriticalInventoryItem merepresentasikan satu baris hasil query Critical Inventory
type CriticalInventoryItem struct {
	Area      string   `json:"area" gorm:"column:area"`
	Sku       string   `json:"sku" gorm:"column:sku"`
	NamaPrdk  string   `json:"nama_produk" gorm:"column:namaPrdk"`
	Bln1      float64  `json:"bln1" gorm:"column:bln1"`
	Bln2      float64  `json:"bln2" gorm:"column:bln2"`
	Bln3      float64  `json:"bln3" gorm:"column:bln3"`
	Bln4      float64  `json:"bln4" gorm:"column:bln4"`
	Bln5      float64  `json:"bln5" gorm:"column:bln5"`
	Bln6      float64  `json:"bln6" gorm:"column:bln6"`
	Avg       float64  `json:"avg" gorm:"column:avg"`
	Stock     float64  `json:"stock" gorm:"column:stock"`
	QtyPo     *float64 `json:"qty_po" gorm:"column:qty_po"`
	Po        *string  `json:"po" gorm:"column:po"`
	PoDate    *string  `json:"po_date" gorm:"column:po_date"`
	Estimasi  *string  `json:"estimasi" gorm:"column:estimasi"`
	Statue    *string  `json:"statue" gorm:"column:statue"`
	Note      string   `json:"note" gorm:"column:note"`
}

// CriticalInventorySummary statistik ringkasan
type CriticalInventorySummary struct {
	TotalItems         int     `json:"total_items"`
	UrgentRestockCount int     `json:"urgent_restock_count"`
	LowStockCount      int     `json:"low_stock_count"`
	CheckCount         int     `json:"check_count"`
	TotalQtyPoPending  float64 `json:"total_qty_po_pending"`
}

// MonthHeader nama-nama bulan untuk header bln1 - bln6
type MonthHeader struct {
	Key   string `json:"key"`   // bln1, bln2, ...
	Label string `json:"label"` // Mei 2026, Jun 2026, ...
}

// SiteStockData merepresentasikan nilai Avg dan Stock per area
type SiteStockData struct {
	Avg   float64 `json:"avg"`
	Stock float64 `json:"stock"`
}

// CriticalInventoryPivotItem merepresentasikan satu baris produk yang dipivot per area dinamis
type CriticalInventoryPivotItem struct {
	Sku           string                   `json:"sku"`
	NamaProduk    string                   `json:"nama_produk"`
	Sites         map[string]SiteStockData `json:"sites"`
	TotalAvg      float64                  `json:"total_avg"`
	TotalStock    float64                  `json:"total_stock"`
	Note          string                   `json:"note"`
	CriticalSites []string                 `json:"critical_sites"`
	QtyPo         *float64                 `json:"qty_po"`
	Po            *string                  `json:"po"`
	PoDate        *string                  `json:"po_date"`
	Estimasi      *string                  `json:"estimasi"`
	Statue        *string                  `json:"statue"`
}

// CriticalInventoryResponse response API lengkap
type CriticalInventoryResponse struct {
	Status       string                       `json:"status"`
	Message      string                       `json:"message,omitempty"`
	SelectedSite string                       `json:"selected_site"`
	Summary      CriticalInventorySummary     `json:"summary"`
	MonthHeaders []MonthHeader                `json:"month_headers"`
	Sites        []string                     `json:"sites"`
	Data         []CriticalInventoryItem      `json:"data"`
	PivotData    []CriticalInventoryPivotItem `json:"pivot_data"`
	LastUpdate   string                       `json:"last_update"`
}

// GetMonthHeadersHelper menghasilkan label nama bulan 6 bulan ke belakang
func GetMonthHeadersHelper(now time.Time) []MonthHeader {
	indonesianMonths := []string{
		"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des",
	}

	// Gunakan tanggal 1 agar aman dari overflow tanggal (seperti tanggal 31)
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	headers := make([]MonthHeader, 6)
	for i := 0; i < 6; i++ {
		// Interval: 5 - i bulan yang lalu (bln1 = 5 bln lalu, bln6 = bulan ini)
		monthsAgo := 5 - i
		t := firstDay.AddDate(0, -monthsAgo, 0)
		monthNum := int(t.Month())
		label := fmt.Sprintf("%s %02d", indonesianMonths[monthNum], t.Year()%100)
		
		headers[i] = MonthHeader{
			Key:   fmt.Sprintf("bln%d", i+1),
			Label: label,
		}
	}
	return headers
}
