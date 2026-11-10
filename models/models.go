package models

import (
	"time"

	"gorm.io/gorm"
)

// ==========================================
// MODELS FOR HO (WORKSHOP DATABASE)
// ==========================================

// Status Unit di Pool / Workshop
type UnitStatus string

const (
	StatusPerbaikan   UnitStatus = "Perbaikan"
	StatusAsuransi    UnitStatus = "Asuransi"
	StatusStandby     UnitStatus = "Standby"
	StatusBongkarMuat UnitStatus = "Bongkar Muat"
	StatusTungguDO    UnitStatus = "Tunggu DO"
)

// Vehicle Model
type Vehicle struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	PlateNumber string         `gorm:"size:20;uniqueIndex;not null" json:"plate_number"`
	Type        string         `gorm:"size:100;not null" json:"type"`     // e.g. Colt Diesel Engkel
	Category    string         `gorm:"size:100;not null" json:"category"` // e.g. Light Truck
	Brand       string         `gorm:"size:100" json:"brand"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Visitor Log (Gate Access)
type VisitorLog struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	VehicleID uint       `gorm:"not null" json:"vehicle_id"`
	Vehicle   Vehicle    `gorm:"foreignKey:VehicleID" json:"vehicle"`
	EntryTime time.Time  `gorm:"not null" json:"entry_time"`
	ExitTime  *time.Time `json:"exit_time,omitempty"`
	Status    UnitStatus `gorm:"size:50;not null" json:"status"` // Perbaikan, Standby, etc.
	Notes     string     `gorm:"type:text" json:"notes"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Service Order
type ServiceOrder struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	OrderNumber string         `gorm:"size:50;uniqueIndex;not null" json:"order_number"`
	VehicleID   uint           `gorm:"not null" json:"vehicle_id"`
	Vehicle     Vehicle        `gorm:"foreignKey:VehicleID" json:"vehicle"`
	Status      UnitStatus     `gorm:"size:50;not null" json:"status"`
	Description string         `gorm:"type:text" json:"description"`
	StartDate   time.Time      `json:"start_date"`
	EndDate     *time.Time     `json:"end_date,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// DTO untuk Dashboard Visitor Response (sesuai UI Sampurna Workshop)
type DashboardVisitorSummary struct {
	TotalUnitInPool   int                     `json:"total_unit_in_pool"`
	PreviousPoolUnit  int                     `json:"previous_pool_unit"`
	Masuk             int                     `json:"masuk_count"`
	Keluar            int                     `json:"keluar_count"`
	StatusBreakdown   map[string]StatusDetail `json:"status_breakdown"`
	VehicleByType     []CategoryCount         `json:"vehicle_by_type"`
	VehicleByCategory []CategoryCount         `json:"vehicle_by_category"`
}

type StatusDetail struct {
	Count         int `json:"count"`
	PreviousCount int `json:"previous_count"`
	Masuk         int `json:"masuk"`
	Keluar        int `json:"keluar"`
}

type CategoryCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DTO untuk Kelengkapan (Service Order Confirmation Summary)
type KelengkapanSummary struct {
	Perbaikan int `gorm:"column:perbaikan" json:"perbaikan"`
	Asuransi  int `gorm:"column:asuransi" json:"asuransi"`
	Storing   int `gorm:"column:storing" json:"storing"`
	Other     int `gorm:"column:other" json:"other"`
	Total     int `gorm:"column:total" json:"total"`
}

// DTO untuk Overdue Summary
type OverdueSummary struct {
	Perbaikan int `gorm:"column:perbaikan" json:"perbaikan"`
	Asuransi  int `gorm:"column:asuransi" json:"asuransi"`
	Storing   int `gorm:"column:storing" json:"storing"`
	Other     int `gorm:"column:other" json:"other"`
	Total     int `gorm:"column:total" json:"total"`
}

// DTO untuk Statistik 6 Kolom Kategori (Visitors Status In/Out/Saldo)
type ColumnCategoryStat struct {
	Kategori  string `gorm:"column:kategori" json:"kategori"`
	LastIn    int    `gorm:"column:last_in" json:"last_in"`
	LastOut   int    `gorm:"column:last_out" json:"last_out"`
	LastSaldo int    `gorm:"column:last_saldo" json:"last_saldo"`
	NowIn     int    `gorm:"column:now_in" json:"now_in"`
	NowOut    int    `gorm:"column:now_out" json:"now_out"`
	NowSaldo  int    `gorm:"column:now_saldo" json:"now_saldo"`
	Total     int    `gorm:"column:total" json:"total"`
}

// DTO untuk Item Kartu Unit per Kategori
type UnitCardItem struct {
	Statue          string `gorm:"column:statue" json:"statue"`
	EntryDate       string `gorm:"column:entryDate" json:"entry_date"`
	Kategori        string `gorm:"column:kategori" json:"kategori"`
	PurposeKategori string `gorm:"column:purpose_kategori" json:"purpose_kategori"`
	Driver          string `gorm:"column:driver" json:"driver"`
	Nopol           string `gorm:"column:nopol" json:"nopol"`
	CustCode        string `gorm:"column:CustCode" json:"cust_code"`
	Estimasi        string `gorm:"column:estimasi" json:"estimasi"`
	IsOverdue       int    `gorm:"column:is_overdue" json:"is_overdue"`
}

// DTO untuk Operational Dashboard
type OperationSummary struct {
	TotalActiveSO    int                           `json:"total_active_so"`
	InRepairCount    int                           `json:"in_repair_count"`
	WaitingPartCount int                           `json:"waiting_part_count"`
	CompletedToday   int                           `json:"completed_today"`
	StoringEmergency int                           `json:"storing_emergency"`
	Kelengkapan      KelengkapanSummary            `json:"kelengkapan"`
	Overdue          OverdueSummary                `json:"overdue"`
	ColumnStats      map[string]ColumnCategoryStat `json:"column_stats"`
	UnitCards        map[string][]UnitCardItem     `json:"unit_cards"`
	ActiveOperations []ActiveOperationItem         `json:"active_operations"`
}

type ActiveOperationItem struct {
	ID           uint   `json:"id"`
	OrderNumber  string `json:"order_number"`
	PlateNumber  string `json:"plate_number"`
	VehicleType  string `json:"vehicle_type"`
	WorkStatus   string `json:"work_status"`
	MechanicName string `json:"mechanic_name"`
	Progress     int    `json:"progress"`
	EntryDate    string `json:"entry_date"`
	EstimatedETA string `json:"estimated_eta"`
}

// DTO untuk Laporan SPK Per Cust
type SpkPerCustItem struct {
	Site     string     `gorm:"column:site" json:"site"`
	TglMasuk *time.Time `gorm:"column:tglMasuk" json:"tgl_masuk"`
	CloseSpk *time.Time `gorm:"column:closeSpk" json:"close_spk"`
	Nopol    string     `gorm:"column:nopol" json:"nopol"`
	CustCode string     `gorm:"column:custCode" json:"cust_code"`
	Kategori string     `gorm:"column:kategori" json:"kategori"`
	Purpose  string     `gorm:"column:purpose" json:"purpose"`
	Spk      string     `gorm:"column:spk" json:"spk"`
	Service  float64    `gorm:"column:service" json:"service"`
	Part     float64    `gorm:"column:part" json:"part"`
	Ban      float64    `gorm:"column:ban" json:"ban"`
	Notes    string     `gorm:"column:notes" json:"notes"`
	OpenSkb  int        `gorm:"column:openskb" json:"openskb"`
}

// DTO untuk Filter Options SPK Per Cust
type AreaOption struct {
	ID   string `gorm:"column:id" json:"id"`
	Site string `gorm:"column:site" json:"site"`
}

type CustomerOption struct {
	ID   string `gorm:"column:id" json:"id"`
	Code string `gorm:"column:code" json:"code"`
}

type SpkFilterOptions struct {
	Areas     []AreaOption     `json:"areas"`
	Customers []CustomerOption `json:"customers"`
	Purposes  []string         `json:"purposes"`
}

// ==========================================
// MODELS FOR CSV DATABASE (imp_sampurna)
// ==========================================

type ImpVisitor struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Nopol       string     `gorm:"column:nopol" json:"nopol"`
	Nobak       string     `gorm:"column:nobak" json:"nobak"`
	Purpose     string     `gorm:"column:purpose" json:"purpose"`
	Information string     `gorm:"column:information" json:"information"`
	Customer    string     `gorm:"column:customer" json:"customer"`
	CustCode    string     `gorm:"column:cust_code" json:"cust_code"`
	Formation   string     `gorm:"column:formation" json:"formation"`
	Planner     string     `gorm:"column:planner" json:"planner"`
	Driver      string     `gorm:"column:driver" json:"driver"`
	CurrDriver  string     `gorm:"column:curr_driver" json:"curr_driver"`
	Tipe        string     `gorm:"column:tipe" json:"tipe"`
	Category    string     `gorm:"column:category" json:"category"`
	EntryDate   *time.Time `gorm:"column:entrydate" json:"entrydate"`
	EntryBy     string     `gorm:"column:entryby" json:"entryby"`
	ExitDate    *time.Time `gorm:"column:exitdate" json:"exitdate"`
	ExitBy      string     `gorm:"column:exitby" json:"exitby"`
	VisitorID   string     `gorm:"column:visitor_id" json:"visitor_id"`
	SiteName    string     `gorm:"column:sitename" json:"sitename"`
	Created     *time.Time `gorm:"column:created" json:"created"`
}

func (ImpVisitor) TableName() string {
	return "imp_visitor"
}

type ImpServiceOrder struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	VisitorID       string     `gorm:"column:visitor_id" json:"visitor_id"`
	SoID            string     `gorm:"column:so_id" json:"so_id"`
	ServiceOrder    string     `gorm:"column:service_order" json:"service_order"`
	Category        string     `gorm:"column:category" json:"category"`
	Reff            string     `gorm:"column:reff" json:"reff"`
	Customer        string     `gorm:"column:customer" json:"customer"`
	CustomerCode    string     `gorm:"column:customer_code" json:"customer_code"`
	Site            string     `gorm:"column:site" json:"site"`
	Information     string     `gorm:"column:information" json:"information"`
	PlateNumber     string     `gorm:"column:plate_number" json:"plate_number"`
	BodyNumber      string     `gorm:"column:body_number" json:"body_number"`
	VehicleType     string     `gorm:"column:vehicle_type" json:"vehicle_type"`
	VehicleCategory string     `gorm:"column:vehicle_category" json:"vehicle_category"`
	Statue          string     `gorm:"column:statue" json:"statue"`
	ReleasedAt      *time.Time `gorm:"column:released_at" json:"released_at"`
	CompletedAt     *time.Time `gorm:"column:completed_at" json:"completed_at"`
	Tipe            string     `gorm:"column:tipe" json:"tipe"`
	Fulfillment     string     `gorm:"column:fulfillment" json:"fulfillment"`
	SKU             string     `gorm:"column:sku" json:"sku"`
	Product         string     `gorm:"column:product" json:"product"`
	Qty             float64    `gorm:"column:qty" json:"qty"`
	UnitPrice       float64    `gorm:"column:unit_price" json:"unit_price"`
	TotalPrice      float64    `gorm:"column:total_price" json:"total_price"`
	Invoice         string     `gorm:"column:invoice" json:"invoice"`
	ServiceOrderID  string     `gorm:"column:service_order_id" json:"service_order_id"`
	LastNotes       string     `gorm:"column:last_notes" json:"last_notes"`
	Created         *time.Time `gorm:"column:created" json:"created"`
}

func (ImpServiceOrder) TableName() string {
	return "imp_service_order"
}
