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
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
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
