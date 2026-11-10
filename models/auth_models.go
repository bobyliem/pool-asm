package models

import "time"

type MenuLogin struct {
	ID        int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Usrn      string     `gorm:"column:usrn" json:"usrn"`
	Pass      string     `gorm:"column:pass" json:"-"`
	Nama      string     `gorm:"column:nama" json:"nama"`
	Aktif     int        `gorm:"column:aktif" json:"aktif"` // 0 = Inactive, 1 = Active
	GrpID     int        `gorm:"column:grp_id" json:"grp_id"`
	LastLogin *time.Time `gorm:"column:last_login" json:"last_login"`
	CreatedBy string     `gorm:"column:createdby" json:"createdby"`
	Created   *time.Time `gorm:"column:created" json:"created"`

	// Relasi Nama Group
	GroupName string `gorm:"->;-:migration;column:group_name" json:"group_name"`
}

func (MenuLogin) TableName() string {
	return "menu_login"
}

type MenuGroup struct {
	ID        int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Nama      string     `gorm:"column:nama" json:"nama"`
	Created   *time.Time `gorm:"column:created" json:"created"`
	UserCount int        `gorm:"->;-:migration;column:user_count" json:"user_count"`
	MenuCount int        `gorm:"->;-:migration;column:menu_count" json:"menu_count"`
}

func (MenuGroup) TableName() string {
	return "menu_group"
}

type MenuMaster struct {
	ID      int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Modul   string     `gorm:"column:modul" json:"modul"`
	Nama    string     `gorm:"column:nama" json:"nama"`
	Urut    string     `gorm:"column:urut" json:"urut"`
	Created *time.Time `gorm:"column:created" json:"created"`
}

func (MenuMaster) TableName() string {
	return "menumaster"
}

type Menu struct {
	ID      int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	MmID    int        `gorm:"column:mm_id" json:"mm_id"`
	Nama    string     `gorm:"column:nama" json:"nama"`
	Urut    string     `gorm:"column:urut" json:"urut"`
	Aktif   int        `gorm:"column:aktif" json:"aktif"`
	Created *time.Time `gorm:"column:created" json:"created"`
}

func (Menu) TableName() string {
	return "menu"
}

type MenuAccess struct {
	ID      int        `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	GrpID   int        `gorm:"column:grp_id" json:"grp_id"`
	MenuID  int        `gorm:"column:menu_id" json:"menu_id"`
	Created *time.Time `gorm:"column:created" json:"created"`
}

func (MenuAccess) TableName() string {
	return "menu_access"
}

// Request & Response DTOs
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      UserDetail `json:"user"`
}

type UserDetail struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Nama      string     `json:"nama"`
	Aktif     int        `json:"aktif"`
	GrpID     int        `json:"grp_id"`
	GroupName string     `json:"group_name"`
	LastLogin *time.Time `json:"last_login"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type CreateUserRequest struct {
	Nama     string `json:"nama" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password"` // Jika kosong, default "123456"
	Aktif    int    `json:"aktif"`    // 0 = Inactive, 1 = Active
	GrpID    int    `json:"grp_id" binding:"required"`
}

type UpdateUserRequest struct {
	Nama     string `json:"nama" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password"` // Jika diisi, ganti password. Jika kosong, pertahankan lama.
	Aktif    int    `json:"aktif"`
	GrpID    int    `json:"grp_id" binding:"required"`
}

type CreateGroupRequest struct {
	Nama string `json:"nama" binding:"required"`
}

type UpdateGroupRequest struct {
	Nama string `json:"nama" binding:"required"`
}

type GroupMenuItem struct {
	MenuID    int    `json:"menu_id"`
	MenuNama  string `json:"menu_nama"`
	MenuUrut  string `json:"menu_urut"`
	MmID      int    `json:"mm_id"`
	Modul     string `json:"modul"`
	ModulNama string `json:"modul_nama"`
	ModulUrut string `json:"modul_urut"`
	Assigned  bool   `json:"assigned"`
}

type SetupGroupMenuRequest struct {
	MenuIDs []int `json:"menu_ids"`
}
