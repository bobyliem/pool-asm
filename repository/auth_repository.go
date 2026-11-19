package repository

import (
	"asm/models"
	"time"

	"gorm.io/gorm"
)

type AuthRepository interface {
	// Autentikasi
	GetUserByUsername(usrn string) (*models.MenuLogin, error)
	GetUserByID(id int) (*models.MenuLogin, error)
	UpdateLastLogin(id int) error
	UpdatePassword(id int, newHash string) error

	// Master User
	GetAllUsers() ([]models.MenuLogin, error)
	CreateUser(u *models.MenuLogin) error
	UpdateUser(u *models.MenuLogin, updatePass bool) error
	DeleteUser(id int) error

	// Master Group
	GetAllGroups() ([]models.MenuGroup, error)
	GetGroupByID(id int) (*models.MenuGroup, error)
	CreateGroup(g *models.MenuGroup) error
	UpdateGroup(g *models.MenuGroup) error
	DeleteGroup(id int) error

	// Setup Group Menu
	GetGroupMenus(grpID int) ([]models.GroupMenuItem, error)
	SaveGroupMenus(grpID int, menuIDs []int) error
	GetAllowedMenuIDs(grpID int) ([]int, error)
	GetUserMenus(grpID int) ([]models.UserMenuItem, error)
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	repo := &authRepository{db: db}
	repo.seedDefaultMenus()
	return repo
}

func (r *authRepository) seedDefaultMenus() {
	if r.db == nil {
		return
	}

	// 1. Pastikan modul ADMINISTRATOR ada di menumaster (modul PRJ-CSV)
	var mm models.MenuMaster
	if err := r.db.Where("id = 9 OR nama = 'ADMINISTRATOR'").First(&mm).Error; err != nil {
		now := time.Now()
		mm = models.MenuMaster{
			ID:      9,
			Modul:   "PRJ-CSV",
			Nama:    "ADMINISTRATOR",
			Urut:    "90",
			Created: &now,
		}
		_ = r.db.Create(&mm).Error
	} else if mm.Modul != "PRJ-CSV" {
		_ = r.db.Model(&models.MenuMaster{}).Where("id = ?", mm.ID).Update("modul", "PRJ-CSV").Error
	}

	// Auto-migrate tabel menu untuk memastikan kolom url dan icon tersedia
	_ = r.db.AutoMigrate(&models.Menu{})

	// 2. Pastikan seluruh default menu ada dan memiliki URL serta icon yang benar
	defaultMenus := []models.Menu{
		{ID: 1, MmID: 1, Nama: "Dashboard", Url: "/prj-csv/dashboard", Icon: "fa-chart-line", Urut: "1", Aktif: 1},
		{ID: 2, MmID: 2, Nama: "SPK Per Cust", Url: "/prj-csv/spk-per-cust", Icon: "fa-file-invoice", Urut: "1", Aktif: 1},
		{ID: 17, MmID: 2, Nama: "SPK /Cust", Url: "/prj-csv/spk-slash-cust", Icon: "fa-file-invoice-dollar", Urut: "2", Aktif: 1},
		{ID: 3, MmID: 2, Nama: "Summary Per Cust", Url: "/prj-csv/summary-per-cust", Icon: "fa-table-cells-large", Urut: "3", Aktif: 1},
		{ID: 4, MmID: 2, Nama: "Resume Per Kategori", Url: "/prj-csv/resume-per-kategori", Icon: "fa-border-all", Urut: "4", Aktif: 1},
		{ID: 5, MmID: 3, Nama: "Resume Per Hari", Url: "/prj-csv/produksi/resume-per-hari", Icon: "fa-calendar-day", Urut: "1", Aktif: 1},
		{ID: 6, MmID: 3, Nama: "Summary Per Driver", Url: "/prj-csv/produksi/summary-per-driver", Icon: "fa-user-tie", Urut: "2", Aktif: 1},
		{ID: 7, MmID: 3, Nama: "Monitoring Ritase", Url: "/prj-csv/produksi/monitoring-ritase", Icon: "fa-table-columns", Urut: "3", Aktif: 1},
		{ID: 16, MmID: 3, Nama: "Transfer Out", Url: "/prj-csv/produksi/transfer-out", Icon: "fa-truck-arrow-right", Urut: "4", Aktif: 1},
		{ID: 8, MmID: 4, Nama: "Pemakaian Solar", Url: "/prj-csv/colokan-solar/pemakaian-solar", Icon: "fa-gas-pump", Urut: "1", Aktif: 1},
		{ID: 19, MmID: 4, Nama: "Import Data", Url: "/prj-csv/colokan-solar/import-data", Icon: "fa-file-import", Urut: "2", Aktif: 1},
		{ID: 20, MmID: 4, Nama: "Laporan Detail Harian", Url: "/prj-csv/colokan-solar/detail-harian", Icon: "fa-calendar-day", Urut: "3", Aktif: 1},
		{ID: 21, MmID: 4, Nama: "Laporan Rekap Bulanan", Url: "/prj-csv/colokan-solar/rekap-unit", Icon: "fa-chart-column", Urut: "4", Aktif: 1},
		{ID: 22, MmID: 4, Nama: "Laporan Rekonsiliasi & Anomali", Url: "/prj-csv/colokan-solar/rekonsiliasi-anomali", Icon: "fa-shield-halved", Urut: "5", Aktif: 1},
		{ID: 9, MmID: 5, Nama: "Prior Unit", Url: "/prj-csv/prior-unit", Icon: "fa-truck-fast", Urut: "1", Aktif: 1},
		{ID: 10, MmID: 5, Nama: "Closing Finance", Url: "/prj-csv/closing-finance", Icon: "fa-file-invoice-dollar", Urut: "2", Aktif: 1},
		{ID: 14, MmID: 9, Nama: "Master User", Url: "/admin/users", Icon: "fa-users-gear", Urut: "1", Aktif: 1},
		{ID: 15, MmID: 9, Nama: "Master Group", Url: "/admin/groups", Icon: "fa-users-rectangle", Urut: "2", Aktif: 1},
		{ID: 18, MmID: 9, Nama: "Cron Jobs", Url: "/admin/cron-jobs", Icon: "fa-clock", Urut: "3", Aktif: 1},
		{ID: 11, MmID: 6, Nama: "Operation Dashboard", Url: "/prj-ho/operation-dashboard", Icon: "fa-gauge-high", Urut: "1", Aktif: 1},
		{ID: 12, MmID: 7, Nama: "SPK Per Cust", Url: "/prj-ho/spk-per-cust", Icon: "fa-file-invoice", Urut: "1", Aktif: 1},
		{ID: 13, MmID: 8, Nama: "Critical Inventory", Url: "/prj-ho/warehouse/critical-inventory", Icon: "fa-boxes-stacked", Urut: "1", Aktif: 1},
	}

	for _, dm := range defaultMenus {
		var existing models.Menu
		if err := r.db.Where("id = ? OR (mm_id = ? AND nama = ?)", dm.ID, dm.MmID, dm.Nama).First(&existing).Error; err != nil {
			now := time.Now()
			dm.Created = &now
			_ = r.db.Create(&dm).Error
		} else {
			// Update url dan icon jika masih kosong
			if existing.Url == "" || existing.Icon == "" {
				_ = r.db.Model(&models.Menu{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
					"url":  dm.Url,
					"icon": dm.Icon,
				}).Error
			}
		}
	}

	// 3. Khusus untuk group SUPERADMIN: Otomatis beri hak akses ke SEMUA menu yang ada di tabel menu
	var superadminGroup models.MenuGroup
	if err := r.db.Where("nama = 'SUPERADMIN' OR id = 1").First(&superadminGroup).Error; err == nil {
		var allMenus []models.Menu
		if err := r.db.Where("aktif = 1").Find(&allMenus).Error; err == nil {
			now := time.Now()
			for _, m := range allMenus {
				var cnt int64
				r.db.Model(&models.MenuAccess{}).Where("grp_id = ? AND menu_id = ?", superadminGroup.ID, m.ID).Count(&cnt)
				if cnt == 0 {
					_ = r.db.Create(&models.MenuAccess{
						GrpID:   superadminGroup.ID,
						MenuID:  m.ID,
						Created: &now,
					}).Error
				}
			}
		}
	}
}

func (r *authRepository) GetUserByUsername(usrn string) (*models.MenuLogin, error) {
	var user models.MenuLogin
	err := r.db.Table("menu_login ml").
		Select("ml.*, COALESCE(mg.nama, '') AS group_name").
		Joins("LEFT JOIN menu_group mg ON ml.grp_id = mg.id").
		Where("ml.usrn = ?", usrn).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) GetUserByID(id int) (*models.MenuLogin, error) {
	var user models.MenuLogin
	err := r.db.Table("menu_login ml").
		Select("ml.*, COALESCE(mg.nama, '') AS group_name").
		Joins("LEFT JOIN menu_group mg ON ml.grp_id = mg.id").
		Where("ml.id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) UpdateLastLogin(id int) error {
	now := time.Now()
	return r.db.Model(&models.MenuLogin{}).Where("id = ?", id).Update("last_login", &now).Error
}

func (r *authRepository) UpdatePassword(id int, newHash string) error {
	return r.db.Model(&models.MenuLogin{}).Where("id = ?", id).Update("pass", newHash).Error
}

func (r *authRepository) GetAllUsers() ([]models.MenuLogin, error) {
	var users []models.MenuLogin
	err := r.db.Table("menu_login ml").
		Select("ml.id, ml.usrn, ml.nama, ml.aktif, ml.grp_id, ml.last_login, ml.createdby, ml.created, COALESCE(mg.nama, '') AS group_name").
		Joins("LEFT JOIN menu_group mg ON ml.grp_id = mg.id").
		Order("ml.id ASC").
		Find(&users).Error
	return users, err
}

func (r *authRepository) CreateUser(u *models.MenuLogin) error {
	return r.db.Create(u).Error
}

func (r *authRepository) UpdateUser(u *models.MenuLogin, updatePass bool) error {
	updates := map[string]interface{}{
		"nama":   u.Nama,
		"usrn":   u.Usrn,
		"aktif":  u.Aktif,
		"grp_id": u.GrpID,
	}
	if updatePass {
		updates["pass"] = u.Pass
	}
	return r.db.Model(&models.MenuLogin{}).Where("id = ?", u.ID).Updates(updates).Error
}

func (r *authRepository) DeleteUser(id int) error {
	return r.db.Delete(&models.MenuLogin{}, id).Error
}

func (r *authRepository) GetAllGroups() ([]models.MenuGroup, error) {
	var groups []models.MenuGroup
	query := `
		SELECT mg.id, mg.nama, mg.created,
		       (SELECT COUNT(*) FROM menu_login ml WHERE ml.grp_id = mg.id) AS user_count,
		       (SELECT COUNT(*) FROM menu_access ma WHERE ma.grp_id = mg.id) AS menu_count
		FROM menu_group mg
		ORDER BY mg.id ASC
	`
	err := r.db.Raw(query).Scan(&groups).Error
	return groups, err
}

func (r *authRepository) GetGroupByID(id int) (*models.MenuGroup, error) {
	var group models.MenuGroup
	query := `
		SELECT mg.id, mg.nama, mg.created,
		       (SELECT COUNT(*) FROM menu_login ml WHERE ml.grp_id = mg.id) AS user_count,
		       (SELECT COUNT(*) FROM menu_access ma WHERE ma.grp_id = mg.id) AS menu_count
		FROM menu_group mg
		WHERE mg.id = ?
	`
	err := r.db.Raw(query, id).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *authRepository) CreateGroup(g *models.MenuGroup) error {
	return r.db.Create(g).Error
}

func (r *authRepository) UpdateGroup(g *models.MenuGroup) error {
	return r.db.Model(&models.MenuGroup{}).Where("id = ?", g.ID).Update("nama", g.Nama).Error
}

func (r *authRepository) DeleteGroup(id int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Hapus menu_access terkait
		if err := tx.Where("grp_id = ?", id).Delete(&models.MenuAccess{}).Error; err != nil {
			return err
		}
		// Hapus group
		return tx.Delete(&models.MenuGroup{}, id).Error
	})
}

func (r *authRepository) GetGroupMenus(grpID int) ([]models.GroupMenuItem, error) {
	var items []models.GroupMenuItem

	query := `
		SELECT 
			m.id AS menu_id,
			m.nama AS menu_nama,
			m.urut AS menu_urut,
			mm.id AS mm_id,
			mm.modul,
			mm.nama AS modul_nama,
			mm.urut AS modul_urut,
			CASE WHEN ma.id IS NOT NULL THEN 1 ELSE 0 END AS assigned
		FROM menu m
		JOIN menumaster mm ON m.mm_id = mm.id
		LEFT JOIN menu_access ma ON ma.menu_id = m.id AND ma.grp_id = ?
		WHERE m.aktif = 1
		ORDER BY mm.modul ASC, CAST(mm.urut AS UNSIGNED) ASC, CAST(m.urut AS UNSIGNED) ASC, m.id ASC
	`

	type rawRow struct {
		MenuID    int    `gorm:"column:menu_id"`
		MenuNama  string `gorm:"column:menu_nama"`
		MenuUrut  string `gorm:"column:menu_urut"`
		MmID      int    `gorm:"column:mm_id"`
		Modul     string `gorm:"column:modul"`
		ModulNama string `gorm:"column:modul_nama"`
		ModulUrut string `gorm:"column:modul_urut"`
		Assigned  int    `gorm:"column:assigned"`
	}

	var rows []rawRow
	if err := r.db.Raw(query, grpID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		items = append(items, models.GroupMenuItem{
			MenuID:    row.MenuID,
			MenuNama:  row.MenuNama,
			MenuUrut:  row.MenuUrut,
			MmID:      row.MmID,
			Modul:     row.Modul,
			ModulNama: row.ModulNama,
			ModulUrut: row.ModulUrut,
			Assigned:  row.Assigned == 1,
		})
	}

	return items, nil
}

func (r *authRepository) SaveGroupMenus(grpID int, menuIDs []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Hapus semua akses menu lama untuk group ini
		if err := tx.Where("grp_id = ?", grpID).Delete(&models.MenuAccess{}).Error; err != nil {
			return err
		}

		// 2. Insert menu_id baru yang dipilih
		if len(menuIDs) > 0 {
			var accesses []models.MenuAccess
			now := time.Now()
			for _, mid := range menuIDs {
				accesses = append(accesses, models.MenuAccess{
					GrpID:   grpID,
					MenuID:  mid,
					Created: &now,
				})
			}
			if err := tx.Create(&accesses).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *authRepository) GetAllowedMenuIDs(grpID int) ([]int, error) {
	var ids []int
	err := r.db.Model(&models.MenuAccess{}).Where("grp_id = ?", grpID).Pluck("menu_id", &ids).Error
	return ids, err
}

func (r *authRepository) GetUserMenus(grpID int) ([]models.UserMenuItem, error) {
	var items []models.UserMenuItem

	query := `
		SELECT 
			m.id AS menu_id,
			m.nama AS menu_nama,
			COALESCE(m.url, '') AS url,
			COALESCE(m.icon, '') AS icon,
			m.urut AS menu_urut,
			mm.id AS mm_id,
			mm.modul,
			mm.nama AS modul_nama,
			mm.urut AS modul_urut
		FROM menu m
		JOIN menumaster mm ON m.mm_id = mm.id
		JOIN menu_access ma ON ma.menu_id = m.id AND ma.grp_id = ?
		WHERE m.aktif = 1
		ORDER BY mm.modul ASC, CAST(mm.urut AS UNSIGNED) ASC, CAST(m.urut AS UNSIGNED) ASC, m.id ASC
	`

	type rawRow struct {
		MenuID    int    `gorm:"column:menu_id"`
		MenuNama  string `gorm:"column:menu_nama"`
		Url       string `gorm:"column:url"`
		Icon      string `gorm:"column:icon"`
		MenuUrut  string `gorm:"column:menu_urut"`
		MmID      int    `gorm:"column:mm_id"`
		Modul     string `gorm:"column:modul"`
		ModulNama string `gorm:"column:modul_nama"`
		ModulUrut string `gorm:"column:modul_urut"`
	}

	var rows []rawRow
	if err := r.db.Raw(query, grpID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		items = append(items, models.UserMenuItem{
			MenuID:    row.MenuID,
			MenuNama:  row.MenuNama,
			Url:       row.Url,
			Icon:      row.Icon,
			MenuUrut:  row.MenuUrut,
			MmID:      row.MmID,
			Modul:     row.Modul,
			ModulNama: row.ModulNama,
			ModulUrut: row.ModulUrut,
		})
	}

	return items, nil
}
