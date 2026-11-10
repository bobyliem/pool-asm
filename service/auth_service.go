package service

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"asm/models"
	"asm/repository"
)

type SessionData struct {
	Token     string
	UserID    int
	Username  string
	Nama      string
	Aktif     int
	GrpID     int
	GroupName string
	ExpiresAt time.Time
}

type SessionStore struct {
	sync.RWMutex
	sessions map[string]*SessionData
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*SessionData),
	}

	// Background cleaner setiap 5 menit untuk membersihkan session expired
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			store.cleanup()
		}
	}()

	return store
}

func (s *SessionStore) cleanup() {
	s.Lock()
	defer s.Unlock()
	now := time.Now()
	for token, sess := range s.sessions {
		if sess.ExpiresAt.Before(now) {
			delete(s.sessions, token)
		}
	}
}

func (s *SessionStore) Create(user *models.MenuLogin) (string, time.Time) {
	s.Lock()
	defer s.Unlock()

	b := make([]byte, 24)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	expiresAt := time.Now().Add(30 * time.Minute)

	s.sessions[token] = &SessionData{
		Token:     token,
		UserID:    user.ID,
		Username:  user.Usrn,
		Nama:      user.Nama,
		Aktif:     user.Aktif,
		GrpID:     user.GrpID,
		GroupName: user.GroupName,
		ExpiresAt: expiresAt,
	}

	return token, expiresAt
}

func (s *SessionStore) Validate(token string) (*SessionData, bool) {
	s.Lock()
	defer s.Unlock()

	sess, exists := s.sessions[token]
	if !exists {
		return nil, false
	}

	now := time.Now()
	if sess.ExpiresAt.Before(now) {
		delete(s.sessions, token)
		return nil, false
	}

	// Sliding expiration: perpanjang 30 menit dari sekarang
	sess.ExpiresAt = now.Add(30 * time.Minute)
	return sess, true
}

func (s *SessionStore) Delete(token string) {
	s.Lock()
	defer s.Unlock()
	delete(s.sessions, token)
}

type AuthService interface {
	HashPassword(password string) string
	Login(req models.LoginRequest) (*models.LoginResponse, error)
	ValidateToken(token string) (*SessionData, bool)
	Logout(token string)
	ChangePassword(userID int, req models.ChangePasswordRequest) error

	// Master User
	GetAllUsers() ([]models.MenuLogin, error)
	GetUserByID(id int) (*models.MenuLogin, error)
	CreateUser(creatorUsername string, req models.CreateUserRequest) error
	UpdateUser(id int, req models.UpdateUserRequest) error
	DeleteUser(id int) error

	// Master Group
	GetAllGroups() ([]models.MenuGroup, error)
	CreateGroup(req models.CreateGroupRequest) error
	UpdateGroup(id int, req models.UpdateGroupRequest) error
	DeleteGroup(id int) error

	// Setup Group Menu
	GetGroupMenus(grpID int) ([]models.GroupMenuItem, error)
	SaveGroupMenus(grpID int, menuIDs []int) error
	GetAllowedMenuIDs(grpID int) ([]int, error)
}

type authService struct {
	repo         repository.AuthRepository
	sessionStore *SessionStore
}

func NewAuthService(repo repository.AuthRepository) AuthService {
	return &authService{
		repo:         repo,
		sessionStore: NewSessionStore(),
	}
}

func (s *authService) HashPassword(password string) string {
	h := md5.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *authService) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	user, err := s.repo.GetUserByUsername(req.Username)
	if err != nil {
		return nil, errors.New("username atau password salah")
	}

	if user.Aktif != 1 {
		return nil, errors.New("akun Anda dinonaktifkan. Silakan hubungi Administrator")
	}

	// Cek Password (MD5 atau fallback plaintext)
	hashedInput := s.HashPassword(req.Password)
	if user.Pass != hashedInput && user.Pass != req.Password {
		return nil, errors.New("username atau password salah")
	}

	// Update last_login
	_ = s.repo.UpdateLastLogin(user.ID)

	// Buat Session 30 Menit
	token, expiresAt := s.sessionStore.Create(user)

	now := time.Now()
	user.LastLogin = &now

	return &models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: models.UserDetail{
			ID:        user.ID,
			Username:  user.Usrn,
			Nama:      user.Nama,
			Aktif:     user.Aktif,
			GrpID:     user.GrpID,
			GroupName: user.GroupName,
			LastLogin: user.LastLogin,
		},
	}, nil
}

func (s *authService) ValidateToken(token string) (*SessionData, bool) {
	sess, valid := s.sessionStore.Validate(token)
	if valid && sess != nil {
		// Update last_login ke DB saat session aktif
		_ = s.repo.UpdateLastLogin(sess.UserID)
	}
	return sess, valid
}

func (s *authService) Logout(token string) {
	s.sessionStore.Delete(token)
}

func (s *authService) ChangePassword(userID int, req models.ChangePasswordRequest) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}

	hashedOld := s.HashPassword(req.OldPassword)
	if user.Pass != hashedOld && user.Pass != req.OldPassword {
		return errors.New("password lama tidak sesuai")
	}

	if len(req.NewPassword) < 4 {
		return errors.New("password baru minimal 4 karakter")
	}

	hashedNew := s.HashPassword(req.NewPassword)
	return s.repo.UpdatePassword(userID, hashedNew)
}

func (s *authService) GetAllUsers() ([]models.MenuLogin, error) {
	return s.repo.GetAllUsers()
}

func (s *authService) GetUserByID(id int) (*models.MenuLogin, error) {
	return s.repo.GetUserByID(id)
}

func (s *authService) CreateUser(creatorUsername string, req models.CreateUserRequest) error {
	// Cek username duplikat
	if existing, _ := s.repo.GetUserByUsername(req.Username); existing != nil {
		return fmt.Errorf("username '%s' sudah digunakan", req.Username)
	}

	pass := req.Password
	if pass == "" {
		pass = "123456" // Default password
	}
	hashedPass := s.HashPassword(pass)

	now := time.Now()
	user := models.MenuLogin{
		Usrn:      req.Username,
		Pass:      hashedPass,
		Nama:      req.Nama,
		Aktif:     req.Aktif,
		GrpID:     req.GrpID,
		CreatedBy: creatorUsername,
		Created:   &now,
	}

	return s.repo.CreateUser(&user)
}

func (s *authService) UpdateUser(id int, req models.UpdateUserRequest) error {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}

	// Cek duplikasi username jika diubah
	if req.Username != user.Usrn {
		if existing, _ := s.repo.GetUserByUsername(req.Username); existing != nil {
			return fmt.Errorf("username '%s' sudah digunakan", req.Username)
		}
	}

	user.Nama = req.Nama
	user.Usrn = req.Username
	user.Aktif = req.Aktif
	user.GrpID = req.GrpID

	updatePass := false
	if req.Password != "" {
		user.Pass = s.HashPassword(req.Password)
		updatePass = true
	}

	return s.repo.UpdateUser(user, updatePass)
}

func (s *authService) DeleteUser(id int) error {
	return s.repo.DeleteUser(id)
}

func (s *authService) GetAllGroups() ([]models.MenuGroup, error) {
	return s.repo.GetAllGroups()
}

func (s *authService) CreateGroup(req models.CreateGroupRequest) error {
	now := time.Now()
	group := models.MenuGroup{
		Nama:    req.Nama,
		Created: &now,
	}
	return s.repo.CreateGroup(&group)
}

func (s *authService) UpdateGroup(id int, req models.UpdateGroupRequest) error {
	group := models.MenuGroup{
		ID:   id,
		Nama: req.Nama,
	}
	return s.repo.UpdateGroup(&group)
}

func (s *authService) DeleteGroup(id int) error {
	return s.repo.DeleteGroup(id)
}

func (s *authService) GetGroupMenus(grpID int) ([]models.GroupMenuItem, error) {
	return s.repo.GetGroupMenus(grpID)
}

func (s *authService) SaveGroupMenus(grpID int, menuIDs []int) error {
	return s.repo.SaveGroupMenus(grpID, menuIDs)
}

func (s *authService) GetAllowedMenuIDs(grpID int) ([]int, error) {
	return s.repo.GetAllowedMenuIDs(grpID)
}
