package models

import "time"

type PriorUnit struct {
	ID       int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Nopol    string     `gorm:"column:nopol" json:"nopol"`
	Cabin    string     `gorm:"column:cabin" json:"cabin"`
	Jenis    string     `gorm:"column:jenis" json:"jenis"`
	Kategori string     `gorm:"column:kategori" json:"kategori"`
	Statue   string     `gorm:"column:statue" json:"statue"` // "0" = Open, "1" = Done
	Note     string     `gorm:"column:note" json:"note"`
	Created  *time.Time `gorm:"column:created" json:"created"`
}

func (PriorUnit) TableName() string {
	return "prior_unit"
}

type PriorUnitRequest struct {
	Nopol    string `json:"nopol" binding:"required"`
	Cabin    string `json:"cabin"`
	Jenis    string `json:"jenis"`
	Kategori string `json:"kategori"` // lelang, asuransi
	Statue   string `json:"statue"`   // 0 = Open, 1 = Done
	Note     string `json:"note"`
}

type PriorUnitSummary struct {
	Total int `json:"total"`
	Open  int `json:"open"`
	Done  int `json:"done"`
}
