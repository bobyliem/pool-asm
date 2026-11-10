package models

type CsvVisitorCategoryItem struct {
	Sitename     string  `gorm:"column:sitename" json:"sitename"`
	Nopol        string  `gorm:"column:nopol" json:"nopol"`
	NoPol        string  `gorm:"column:no_pol" json:"no_pol"`
	CustCode     string  `gorm:"column:cust_code" json:"cust_code"`
	Driver       string  `gorm:"column:driver" json:"driver"`
	Category     string  `gorm:"column:category" json:"category"`
	TglMasuk     string  `gorm:"column:tglmasuk" json:"tglmasuk"`
	TglMasukDate string  `gorm:"column:tgl_masuk" json:"tgl_masuk"`
	Note     string  `gorm:"column:note" json:"note"`
	NoteLate string  `gorm:"column:notelate" json:"notelate"`
	Catt     string  `gorm:"column:catt" json:"catt"`
	Exitdate   *string `gorm:"column:exitdate" json:"exitdate"`
	Purpose    string  `gorm:"column:purpose" json:"purpose"`
	IsNotToday int     `gorm:"column:is_not_today" json:"is_not_today"`
}

type CsvCategoryStat struct {
	Purpose   string `gorm:"column:purpose" json:"purpose"`
	LastIn    int    `gorm:"column:last_in" json:"last_in"`
	LastOut   int    `gorm:"column:last_out" json:"last_out"`
	LastIdle  int    `gorm:"column:last_idle" json:"last_idle"`
	TodayIn   int    `gorm:"column:today_in" json:"today_in"`
	TodayOut  int    `gorm:"column:today_out" json:"today_out"`
	TodayIdle int    `gorm:"column:today_idle" json:"today_idle"`
	TotalIdle int    `json:"total_idle"`
}

type CsvDashboardDataResponse struct {
	Areas      []string                            `json:"areas"`
	Counts     map[string]int                      `json:"counts"`
	Stats      map[string]CsvCategoryStat          `json:"stats"`
	Categories map[string][]CsvVisitorCategoryItem `json:"categories"`
	LastUpdate string                              `json:"last_update"`
}
