package controllers

import (
	"KPI_System_Backend/helper" // Pastikan import ini ada
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

type loginAttemptInfo struct {
	Count        int
	FirstAttempt time.Time
	BlockedUntil time.Time
}

var (
	failedLoginMu         sync.Mutex
	failedLoginAttempts   = make(map[string]*loginAttemptInfo)
	failedAuditTimestamps = make(map[string]time.Time)
	maxFailedAttempts     = 5
	attemptWindow         = 1 * time.Hour
	blockDuration         = 1 * time.Hour
	auditSuppressWindow   = 1 * time.Minute
)

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

	ip := c.ClientIP()
	if blocked, message := ac.checkLoginBlock(ip); blocked {
		helper.LogActivityWithUsername(ac.DB, 0, loginReq.Username, "LOGIN_BLOCKED", message, ip)
		Response(c, http.StatusTooManyRequests, message, nil)
		return
	}

	// 1. Cari user hanya berdasarkan username (Ubah query pencarian)
	var user models.User
	if err := ac.DB.Preload("Employee").Where("username = ?", loginReq.Username).First(&user).Error; err != nil {
		logger.Warn("Login attempt for non-existent user", zap.String("username", loginReq.Username), zap.String("ip", ip))
		ac.recordFailedLogin(ip)
		ac.maybeLogFailedAttempt(0, loginReq.Username, "LOGIN_FAILED", fmt.Sprintf("Login gagal untuk username '%s'", loginReq.Username), ip)
		Response(c, http.StatusUnauthorized, "Username atau password salah", nil)
		return
	}

	// 2. CEK STATUS AKTIF (BLOKIR DO) <-- LOGIKA BARU UNTUK FITUR DO
	if !user.IsActive {
		logger.Warn("Login attempt by inactive user", zap.String("username", user.Username), zap.String("ip", ip))
		ac.maybeLogFailedAttempt(user.ID, loginReq.Username, "LOGIN_FAILED", "Akun tidak aktif", ip)
		Response(c, http.StatusForbidden, "Akses Ditolak. Akun Anda telah dinonaktifkan oleh HRD.", nil)
		return
	}

	// 3. Check password
	if !helper.CheckPasswordHash(loginReq.Password, user.PasswordHash) {
		logger.Warn("Invalid password attempt", zap.String("username", loginReq.Username), zap.String("ip", ip))
		ac.recordFailedLogin(ip)
		ac.maybeLogFailedAttempt(user.ID, loginReq.Username, "LOGIN_FAILED", "Password salah", ip)
		Response(c, http.StatusUnauthorized, "Username atau password salah", nil)
		return
	}

	// Reset failed counter on successful login
	ac.resetFailedLogin(ip)

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
	helper.LogActivityWithUsername(ac.DB, user.ID, loginReq.Username, "LOGIN", "User berhasil login", ip)

	loginResp := models.LoginResponse{
		Token: token,
		User:  user,
	}

	logger.Info("User logged in successfully", zap.String("username", user.Username), zap.String("role", user.Role), zap.String("ip", ip))
	Response(c, http.StatusOK, "Login successful", loginResp)
}

func (ac *AuthController) checkLoginBlock(ip string) (bool, string) {
	failedLoginMu.Lock()
	defer failedLoginMu.Unlock()

	attempt, exists := failedLoginAttempts[ip]
	if !exists {
		return false, ""
	}

	now := time.Now()
	if attempt.BlockedUntil.After(now) {
		remaining := attempt.BlockedUntil.Sub(now).Round(time.Second)
		return true, fmt.Sprintf("Terlalu banyak percobaan login. Coba lagi setelah %s", remaining)
	}

	if now.Sub(attempt.FirstAttempt) > attemptWindow {
		delete(failedLoginAttempts, ip)
		return false, ""
	}

	return false, ""
}

func (ac *AuthController) recordFailedLogin(ip string) {
	failedLoginMu.Lock()
	defer failedLoginMu.Unlock()

	now := time.Now()
	attempt, exists := failedLoginAttempts[ip]
	if !exists {
		failedLoginAttempts[ip] = &loginAttemptInfo{Count: 1, FirstAttempt: now}
		return
	}

	if attempt.BlockedUntil.After(now) {
		return
	}

	if now.Sub(attempt.FirstAttempt) > attemptWindow {
		attempt.Count = 1
		attempt.FirstAttempt = now
		attempt.BlockedUntil = time.Time{}
		return
	}

	attempt.Count++
	if attempt.Count >= maxFailedAttempts {
		attempt.BlockedUntil = now.Add(blockDuration)
	}
}

func (ac *AuthController) maybeLogFailedAttempt(userID uint, username, action, description, ip string) {
	// Use IP + action only so audit entries from the same IP are grouped as one actor,
	// even when the attempted username differs.
	key := fmt.Sprintf("%s|%s", ip, action)
	failedLoginMu.Lock()
	defer failedLoginMu.Unlock()

	now := time.Now()
	lastLogged, exists := failedAuditTimestamps[key]
	if exists && now.Sub(lastLogged) < auditSuppressWindow {
		return
	}

	failedAuditTimestamps[key] = now
	helper.LogActivityWithUsername(ac.DB, userID, username, action, description, ip)
}

func (ac *AuthController) resetFailedLogin(ip string) {
	failedLoginMu.Lock()
	defer failedLoginMu.Unlock()
	delete(failedLoginAttempts, ip)
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
