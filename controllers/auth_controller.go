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
	attemptWindow         = 24 * time.Hour
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

	// [AUTO-NOTIFICATION] Jika pegawai belum melengkapi No. Telepon
	if user.EmployeeID != 0 && user.Employee.Phone == "" {
		var count int64
		ac.DB.Model(&models.Notification{}).Where("user_id = ? AND type = ? AND is_read = ?", user.ID, "profile", false).Count(&count)
		if count == 0 {
			notif := models.Notification{
				UserID:  user.ID,
				Title:   "Lengkapi No. Telepon Anda",
				Message: "Harap segera lengkapi No. Telepon/WhatsApp di menu Profil Saya agar mudah dihubungi oleh atasan atau Admin.",
				Type:    "profile",
				IsRead:  false,
			}
			ac.DB.Create(&notif)
		}
	}

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
		var extraBlock time.Duration
		switch attempt.Count {
		case 5:
			extraBlock = 5 * time.Minute
		case 6:
			extraBlock = 10 * time.Minute
		case 7:
			extraBlock = 30 * time.Minute
		case 8:
			extraBlock = 1 * time.Hour
		default: // 9 atau lebih
			extraBlock = 24 * time.Hour
		}
		attempt.BlockedUntil = now.Add(extraBlock)
	}
}

func (ac *AuthController) maybeLogFailedAttempt(userID uint, username, action, description, ip string) {
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

	Response(c, http.StatusOK, "Profile retrieved successfully", user)
}

type UpdateBiodataRequest struct {
	Phone                 string `json:"phone"`
	Bio                   string `json:"bio"`
	SocialMedia           string `json:"social_media"`
	Address               string `json:"address"`
	BirthPlace            string `json:"birth_place"`
	BirthDate             string `json:"birth_date"`
	Gender                string `json:"gender"`
	Education             string `json:"education"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
}

func (ac *AuthController) UpdateBiodata(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req UpdateBiodataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format biodata tidak valid", nil)
		return
	}

	var user models.User
	if err := ac.DB.First(&user, userID).Error; err != nil {
		Response(c, http.StatusNotFound, "User tidak ditemukan", nil)
		return
	}

	if user.EmployeeID == 0 {
		Response(c, http.StatusBadRequest, "Akun Anda tidak terhubung ke data pegawai", nil)
		return
	}

	var emp models.Employee
	if err := ac.DB.First(&emp, user.EmployeeID).Error; err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	isPhoneChanged := emp.Phone != req.Phone

	updates := map[string]interface{}{
		"phone":                   req.Phone,
		"bio":                     req.Bio,
		"social_media":            req.SocialMedia,
		"address":                 req.Address,
		"birth_place":             req.BirthPlace,
		"birth_date":              req.BirthDate,
		"gender":                  req.Gender,
		"education":               req.Education,
		"emergency_contact_name":  req.EmergencyContactName,
		"emergency_contact_phone": req.EmergencyContactPhone,
	}

	if isPhoneChanged {
		updates["is_phone_verified"] = false
		updates["phone_otp"] = ""
		updates["phone_otp_expired_at"] = nil
	}

	if err := ac.DB.Model(&emp).Updates(updates).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memperbarui biodata", nil)
		return
	}

	// [CLEAR NOTIFICATION] Jika phone sudah diisi, otomatis tandai notifikasi profil sebagai read
	if req.Phone != "" {
		ac.DB.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, "profile").Update("is_read", true)
	}

	if idUint, ok := userID.(uint); ok {
		helper.LogActivity(ac.DB, idUint, "UPDATE_BIODATA", "User memperbarui biodata dan nomor telepon", c.ClientIP())
	}

	var updatedUser models.User
	ac.DB.Preload("Employee").Preload("Employee.Division").First(&updatedUser, userID)

	Response(c, http.StatusOK, "Biodata berhasil diperbarui", updatedUser)
}

func (ac *AuthController) SendPhoneOTP(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var user models.User
	if err := ac.DB.Preload("Employee").First(&user, userID).Error; err != nil {
		Response(c, http.StatusNotFound, "User tidak ditemukan", nil)
		return
	}

	if user.EmployeeID == 0 {
		Response(c, http.StatusBadRequest, "Akun Anda tidak terhubung dengan data pegawai", nil)
		return
	}

	if user.Employee.Phone == "" {
		Response(c, http.StatusBadRequest, "Silakan lengkapi nomor telepon terlebih dahulu", nil)
		return
	}

	otpCode, err := helper.GenerateOTP()
	if err != nil {
		Response(c, http.StatusInternalServerError, "Gagal membuat OTP", nil)
		return
	}

	expiry := time.Now().Add(5 * time.Minute)
	
	if err := ac.DB.Model(&models.Employee{}).Where("id = ?", user.EmployeeID).Updates(map[string]interface{}{
		"phone_otp":            otpCode,
		"phone_otp_expired_at": &expiry,
		"is_phone_verified":   false,
	}).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan OTP", nil)
		return
	}

	// Kirim OTP via WhatsApp (utama), dengan fallback ke email jika WA belum siap
	go func(phone, otp, name, email string) {
		if helper.IsWhatsAppReady() {
			err := helper.SendPhoneOTPWhatsApp(phone, otp, name)
			if err != nil {
				// Fallback ke email jika WhatsApp gagal
				_ = helper.SendPhoneOTPEmail(email, otp, phone)
			}
		} else {
			// WhatsApp belum siap, kirim via email sebagai fallback
			_ = helper.SendPhoneOTPEmail(email, otp, phone)
		}
	}(user.Employee.Phone, otpCode, user.Employee.Name, user.Employee.Email)

	if idUint, ok := userID.(uint); ok {
		helper.LogActivity(ac.DB, idUint, "SEND_PHONE_OTP", "Mengirim kode OTP verifikasi nomor telepon via WhatsApp", c.ClientIP())
	}

	if helper.IsWhatsAppReady() {
		Response(c, http.StatusOK, "Kode verifikasi OTP berhasil dikirim ke WhatsApp Anda", nil)
	} else {
		Response(c, http.StatusOK, "WhatsApp bot belum terhubung. Kode OTP dikirim ke email Anda sebagai alternatif", nil)
	}
}

type VerifyPhoneOTPRequest struct {
	OTP string `json:"otp" binding:"required,len=6"`
}

func (ac *AuthController) VerifyPhoneOTP(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req VerifyPhoneOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format OTP tidak valid. Harus 6 digit.", nil)
		return
	}

	var user models.User
	if err := ac.DB.Preload("Employee").First(&user, userID).Error; err != nil {
		Response(c, http.StatusNotFound, "User tidak ditemukan", nil)
		return
	}

	if user.EmployeeID == 0 {
		Response(c, http.StatusBadRequest, "Akun Anda tidak terhubung dengan data pegawai", nil)
		return
	}

	if user.Employee.PhoneOTP == "" || user.Employee.PhoneOTPExpiredAt == nil {
		Response(c, http.StatusBadRequest, "Tidak ada permintaan OTP aktif atau OTP sudah kedaluwarsa", nil)
		return
	}

	if time.Now().After(*user.Employee.PhoneOTPExpiredAt) {
		Response(c, http.StatusBadRequest, "Kode OTP sudah kedaluwarsa. Silakan minta kode baru.", nil)
		return
	}

	if user.Employee.PhoneOTP != req.OTP {
		Response(c, http.StatusBadRequest, "Kode OTP yang Anda masukkan salah", nil)
		return
	}

	if err := ac.DB.Model(&models.Employee{}).Where("id = ?", user.EmployeeID).Updates(map[string]interface{}{
		"phone_otp":            "",
		"phone_otp_expired_at": nil,
		"is_phone_verified":   true,
	}).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memperbarui status verifikasi", nil)
		return
	}

	if idUint, ok := userID.(uint); ok {
		helper.LogActivity(ac.DB, idUint, "VERIFY_PHONE_OTP", "Berhasil memverifikasi nomor telepon", c.ClientIP())
	}

	Response(c, http.StatusOK, "Nomor telepon berhasil diverifikasi!", nil)
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (ac *AuthController) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format password tidak valid (min 6 karakter)", nil)
		return
	}

	var user models.User
	if err := ac.DB.First(&user, userID).Error; err != nil {
		Response(c, http.StatusNotFound, "User tidak ditemukan", nil)
		return
	}

	if !helper.CheckPasswordHash(req.OldPassword, user.PasswordHash) {
		Response(c, http.StatusBadRequest, "Password lama salah", nil)
		return
	}

	newHash, err := helper.HashPassword(req.NewPassword)
	if err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memproses password baru", nil)
		return
	}

	if err := ac.DB.Model(&user).Update("password_hash", newHash).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengupdate password", nil)
		return
	}

	if idUint, ok := userID.(uint); ok {
		helper.LogActivity(ac.DB, idUint, "CHANGE_PASSWORD", "User mengubah password", c.ClientIP())
	}

	Response(c, http.StatusOK, "Password berhasil diubah", nil)
}

func (ac *AuthController) Logout(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusOK, "Logout berhasil", nil)
		return
	}

	var user models.User
	if err := ac.DB.First(&user, userID).Error; err == nil {
		helper.LogActivityWithUsername(ac.DB, user.ID, user.Username, "LOGOUT", "User keluar dari sistem (Logout)", c.ClientIP())
	}

	Response(c, http.StatusOK, "Logout berhasil", nil)
}
