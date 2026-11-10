package handler

import (
	"asm/models"
	"asm/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Helper untuk ekstrak token dari cookie atau Authorization Header
func (h *AuthHandler) extractToken(c *gin.Context) string {
	// 1. Cek dari Cookie
	if cookie, err := c.Cookie("asm_session"); err == nil && cookie != "" {
		return cookie
	}
	// 2. Cek dari Header Authorization: Bearer <token>
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	// 3. Cek dari query param ?token=
	return c.Query("token")
}

// Middleware Autentikasi
func (h *AuthHandler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := h.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Sesi tidak ditemukan. Silakan login kembali.",
			})
			c.Abort()
			return
		}

		sess, valid := h.authService.ValidateToken(token)
		if !valid || sess == nil {
			// Hapus cookie kadaluarsa
			c.SetCookie("asm_session", "", -1, "/", "", false, false)
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Sesi Anda telah berakhir (30 menit). Silakan login kembali.",
			})
			c.Abort()
			return
		}

		// Perbarui cookie sliding window 30 menit (1800 detik)
		c.SetCookie("asm_session", token, 1800, "/", "", false, false)

		// Set context user info
		c.Set("user_id", sess.UserID)
		c.Set("username", sess.Username)
		c.Set("nama", sess.Nama)
		c.Set("grp_id", sess.GrpID)
		c.Set("group_name", sess.GroupName)

		c.Next()
	}
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Username dan password harus diisi",
		})
		return
	}

	resp, err := h.authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Set Cookie Session 30 Menit (1800 detik)
	c.SetCookie("asm_session", resp.Token, 1800, "/", "", false, false)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Login berhasil",
		"data":    resp,
	})
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	token := h.extractToken(c)
	if token != "" {
		h.authService.Logout(token)
	}
	c.SetCookie("asm_session", "", -1, "/", "", false, false)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logout berhasil",
	})
}

// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	token := h.extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Sesi tidak ditemukan",
		})
		return
	}

	sess, valid := h.authService.ValidateToken(token)
	if !valid || sess == nil {
		c.SetCookie("asm_session", "", -1, "/", "", false, false)
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Sesi telah berakhir",
		})
		return
	}

	// Perbarui cookie sliding window 30 menit
	c.SetCookie("asm_session", token, 1800, "/", "", false, false)

	// Ambil data user realtime dari database agar jika ada perubahan nama/group langsung terupdate
	freshUser, err := h.authService.GetUserByID(sess.UserID)
	if err == nil && freshUser != nil {
		sess.Nama = freshUser.Nama
		sess.Username = freshUser.Usrn
		sess.GrpID = freshUser.GrpID
		sess.GroupName = freshUser.GroupName
		sess.Aktif = freshUser.Aktif
	}

	// Ambil daftar menu yang diizinkan untuk group user ini
	allowedMenuIDs, _ := h.authService.GetAllowedMenuIDs(sess.GrpID)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"id":               sess.UserID,
			"username":         sess.Username,
			"nama":             sess.Nama,
			"aktif":            sess.Aktif,
			"grp_id":           sess.GrpID,
			"group_name":       sess.GroupName,
			"expires_at":       sess.ExpiresAt,
			"allowed_menu_ids": allowedMenuIDs,
		},
	})
}

// POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
		})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data tidak lengkap: " + err.Error(),
		})
		return
	}

	if err := h.authService.ChangePassword(userID.(int), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password berhasil diubah",
	})
}

// ==========================================
// MASTER USER CRUD HANDLERS
// ==========================================

// GET /api/v1/admin/users
func (h *AuthHandler) GetAllUsers(c *gin.Context) {
	users, err := h.authService.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   users,
	})
}

// POST /api/v1/admin/users
func (h *AuthHandler) CreateUser(c *gin.Context) {
	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)
	if creatorStr == "" {
		creatorStr = "system"
	}

	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data user tidak valid: " + err.Error(),
		})
		return
	}

	if err := h.authService.CreateUser(creatorStr, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User berhasil ditambahkan",
	})
}

// PUT /api/v1/admin/users/:id
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data update tidak valid: " + err.Error(),
		})
		return
	}

	if err := h.authService.UpdateUser(id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User berhasil diperbarui",
	})
}

// DELETE /api/v1/admin/users/:id
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	// Cegah hapus diri sendiri
	curUserID, _ := c.Get("user_id")
	if curUserID == id {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Tidak dapat menghapus akun Anda sendiri"})
		return
	}

	if err := h.authService.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menghapus user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User berhasil dihapus",
	})
}

// ==========================================
// MASTER GROUP CRUD & SETUP MENU HANDLERS
// ==========================================

// GET /api/v1/admin/groups
func (h *AuthHandler) GetAllGroups(c *gin.Context) {
	groups, err := h.authService.GetAllGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data group: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   groups,
	})
}

// POST /api/v1/admin/groups
func (h *AuthHandler) CreateGroup(c *gin.Context) {
	var req models.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Nama group harus diisi",
		})
		return
	}

	if err := h.authService.CreateGroup(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menambahkan group: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Group berhasil ditambahkan",
	})
}

// PUT /api/v1/admin/groups/:id
func (h *AuthHandler) UpdateGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	var req models.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Nama group harus diisi",
		})
		return
	}

	if err := h.authService.UpdateGroup(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengubah group: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Group berhasil diperbarui",
	})
}

// DELETE /api/v1/admin/groups/:id
func (h *AuthHandler) DeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	// Cegah hapus group default
	if id == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Group SUPERADMIN tidak dapat dihapus"})
		return
	}

	if err := h.authService.DeleteGroup(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menghapus group: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Group dan hak aksesnya berhasil dihapus",
	})
}

// GET /api/v1/admin/groups/:id/menus
func (h *AuthHandler) GetGroupMenus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	menus, err := h.authService.GetGroupMenus(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil daftar menu: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   menus,
	})
}

// POST /api/v1/admin/groups/:id/menus
func (h *AuthHandler) SaveGroupMenus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ID tidak valid"})
		return
	}

	var req models.SetupGroupMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data menu tidak valid: " + err.Error(),
		})
		return
	}

	if err := h.authService.SaveGroupMenus(id, req.MenuIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menyimpan hak akses menu: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Hak akses menu group berhasil diperbarui",
	})
}
