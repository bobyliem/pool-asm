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
