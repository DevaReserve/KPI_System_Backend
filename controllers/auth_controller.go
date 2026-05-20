package controllers

import (
	"net/http"
	"time"
	"KPI_System_Backend/helper" // Pastikan import ini ada
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models"

	"go.uber.org/zap"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{DB: db}
}

func (ac *AuthController) Login(c *gin.Context) {
    var loginReq models.LoginRequest
    if err := c.ShouldBindJSON(&loginReq); err != nil {
        logger.Error("Invalid login request", zap.Error(err))
        Response(c, http.StatusBadRequest, "Invalid request format", nil)
        return
    }

    // 1. Cari user hanya berdasarkan username (Ubah query pencarian)
    var user models.User
    if err := ac.DB.Preload("Employee").Where("username = ?", loginReq.Username).First(&user).Error; err != nil {
        logger.Warn("Login attempt for non-existent user", zap.String("username", loginReq.Username))
        Response(c, http.StatusUnauthorized, "Username atau password salah", nil)
        return
    }

    // 2. CEK STATUS AKTIF (BLOKIR DO) <-- LOGIKA BARU UNTUK FITUR DO
    if !user.IsActive {
        logger.Warn("Login attempt by inactive user", zap.String("username", user.Username))
        Response(c, http.StatusForbidden, "Akses Ditolak. Akun Anda telah dinonaktifkan oleh HRD.", nil)
        return
    }

    // 3. Check password
    if !helper.CheckPasswordHash(loginReq.Password, user.PasswordHash) {
        logger.Warn("Invalid password attempt", zap.String("username", loginReq.Username))
        Response(c, http.StatusUnauthorized, "Username atau password salah", nil)
        return
    }

    // 4. Generate JWT token (Tambahkan user.IsExecutive di belakang) <-- LOGIKA BARU CEO
    token, err := helper.GenerateJWT(user.ID, user.Username, user.Role, user.IsExecutive)
    if err != nil {
        logger.Error("Failed to generate JWT token", zap.Error(err))
        Response(c, http.StatusInternalServerError, "Login failed", nil)
        return
    }

    // Update last login
    ac.DB.Model(&user).Update("last_login", time.Now())

    // --- [AUDIT TRAIL] LOG ACTIVITY ---
    helper.LogActivity(ac.DB, user.ID, "LOGIN", "User berhasil login", c.ClientIP())

    loginResp := models.LoginResponse{
        Token: token,
        User:  user,
    }

    logger.Info("User logged in successfully", zap.String("username", user.Username), zap.String("role", user.Role))
    Response(c, http.StatusOK, "Login successful", loginResp)
}


func (ac *AuthController) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var user models.User
	if err := ac.DB.Preload("Employee").Preload("Employee.Division").First(&user, userID).Error; err != nil {
		Response(c, http.StatusNotFound, "User not found", nil)
		return
	}

    // [ANTI-SPAM]: Tidak perlu log untuk Read (Get Profile)
	Response(c, http.StatusOK, "Profile retrieved successfully", user)
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (ac *AuthController) ChangePassword(c *gin.Context) {
	// 1. Ambil UserID dari Token
	userID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// 2. Bind Request
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format password tidak valid (min 6 karakter)", nil)
		return
	}

	// 3. Cari User di Database
	var user models.User
	if err := ac.DB.First(&user, userID).Error; err != nil {
		Response(c, http.StatusNotFound, "User tidak ditemukan", nil)
		return
	}

	// 4. Verifikasi Password Lama
	if !helper.CheckPasswordHash(req.OldPassword, user.PasswordHash) {
		Response(c, http.StatusBadRequest, "Password lama salah", nil)
		return
	}

	// 5. Hash Password Baru
	newHash, err := helper.HashPassword(req.NewPassword)
	if err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memproses password baru", nil)
		return
	}

	// 6. Simpan Password Baru
	if err := ac.DB.Model(&user).Update("password_hash", newHash).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengupdate password", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    // Aksi sensitif seperti ganti password wajib dicatat
    if idUint, ok := userID.(uint); ok {
	    helper.LogActivity(ac.DB, idUint, "CHANGE_PASSWORD", "User mengubah password", c.ClientIP())
    }

	Response(c, http.StatusOK, "Password berhasil diubah", nil)
}