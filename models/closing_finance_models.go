package models

import "time"

type ExtClosing struct {
	ID           int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Cust         string     `gorm:"column:cust" json:"cust"`
	CustCode     string     `gorm:"column:cust_code" json:"cust_code"`
	ServiceOrder string     `gorm:"column:service_order" json:"service_order"`
	Category     string     `gorm:"column:category" json:"category"`
	Reff         string     `gorm:"column:reff" json:"reff"`
	Nopol        string     `gorm:"column:nopol" json:"nopol"`
	Information  string     `gorm:"column:information" json:"information"`
	Released     *time.Time `gorm:"column:released" json:"released"`
	Completed    *time.Time `gorm:"column:completed" json:"completed"`
	Tipe         string     `gorm:"column:tipe" json:"tipe"`
	Sku          string     `gorm:"column:sku" json:"sku"`
	Prd          string     `gorm:"column:prd" json:"prd"`
	Notes        string     `gorm:"column:notes" json:"notes"`
	Qty          float64    `gorm:"column:qty" json:"qty"`
	Average      float64    `gorm:"column:average" json:"average"`
	Harga        float64    `gorm:"column:harga" json:"harga"`
	Total        float64    `gorm:"column:total" json:"total"`
	Statue       string     `gorm:"column:statue" json:"statue"`
	Created      *time.Time `gorm:"column:created" json:"created"`
}

func (ExtClosing) TableName() string {
	return "ext_closing"
}

type ClosingFinanceFilter struct {
	Page     int    `form:"page,default=1"`
	Limit    int    `form:"limit,default=20"`
	Search   string `form:"search"`
	Customer string `form:"customer"`
	Category string `form:"category"`
	Tipe     string `form:"tipe"`
	TglAwal  string `form:"tglawal"`
	TglAkhir string `form:"tglakhir"`
}

type ClosingFinanceSummary struct {
	TotalRows     int64   `json:"total_rows"`
	TotalSPK      int64   `json:"total_spk"`
	TotalUnit     int64   `json:"total_unit"`
	TotalCustomer int64   `json:"total_customer"`
	GrandTotal    float64 `json:"grand_total"`
}
