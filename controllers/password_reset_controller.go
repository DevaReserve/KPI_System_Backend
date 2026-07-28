package controllers

import (
	"KPI_System_Backend/helper"
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PasswordResetController struct {
	DB *gorm.DB
}

func NewPasswordResetController(db *gorm.DB) *PasswordResetController {
	return &PasswordResetController{DB: db}
}

// ===================================================================
// REQUEST STRUCTS
// ===================================================================

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type ResetPasswordRequest struct {
	Email           string `json:"email" binding:"required,email"`
	OTP             string `json:"otp" binding:"required,len=6"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=6"`
}

// ===================================================================
// ENDPOINT 1: POST /api/auth/forgot-password
// ===================================================================
// Menerima email, generate OTP, simpan ke DB (hashed), kirim via email.
// Response selalu generik agar tidak bocorkan informasi apakah email terdaftar.
func (pc *PasswordResetController) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format email tidak valid", nil)
		return
	}

	// 1. Cek apakah email terdaftar di tabel employees
	var employee models.Employee
	if err := pc.DB.Where("email = ?", req.Email).First(&employee).Error; err != nil {
		// SECURITY: Tetap kembalikan response sukses agar attacker
		// tidak bisa enumerasi email mana yang terdaftar
		logger.Warn("Forgot password request for unregistered email", zap.String("email", req.Email))
		Response(c, http.StatusOK, "Jika email terdaftar, kode OTP telah dikirim ke email Anda", nil)
		return
	}

	// 2. Cek apakah user (akun login) ada untuk employee ini
	var user models.User
	if err := pc.DB.Where("employee_id = ?", employee.ID).First(&user).Error; err != nil {
		logger.Warn("Forgot password: employee found but no user account", zap.String("email", req.Email))
		Response(c, http.StatusOK, "Jika email terdaftar, kode OTP telah dikirim ke email Anda", nil)
		return
	}

	// 3. Invalidasi semua OTP sebelumnya untuk email ini (mencegah OTP lama masih aktif)
	pc.DB.Model(&models.PasswordReset{}).
		Where("email = ? AND is_used = ?", req.Email, false).
		Update("is_used", true)

	// 4. Generate 6 digit OTP
	otpCode, err := helper.GenerateOTP()
	if err != nil {
		logger.Error("Failed to generate OTP", zap.Error(err))
		Response(c, http.StatusInternalServerError, "Gagal memproses permintaan", nil)
		return
	}

	// 5. Hash OTP sebelum simpan ke database (security best practice)
	otpHash, err := helper.HashPassword(otpCode)
	if err != nil {
		logger.Error("Failed to hash OTP", zap.Error(err))
		Response(c, http.StatusInternalServerError, "Gagal memproses permintaan", nil)
		return
	}

	// 6. Simpan record PasswordReset ke database
	passwordReset := models.PasswordReset{
		Email:     req.Email,
		OTPHash:   otpHash,
		ExpiresAt: time.Now().Add(10 * time.Minute), // OTP berlaku 10 menit
		IsUsed:    false,
	}
	if err := pc.DB.Create(&passwordReset).Error; err != nil {
		logger.Error("Failed to save password reset record", zap.Error(err))
		Response(c, http.StatusInternalServerError, "Gagal memproses permintaan", nil)
		return
	}

	// 7. Kirim OTP via email (dalam goroutine agar response cepat)
	go func(email, otp string) {
		if err := helper.SendOTPEmail(email, otp); err != nil {
			logger.Error("Failed to send OTP email", zap.String("email", email), zap.Error(err))
		}
	}(req.Email, otpCode)

	// 8. Log activity
	helper.LogActivity(pc.DB, user.ID, "FORGOT_PASSWORD", "Permintaan reset password via OTP", c.ClientIP())

	logger.Info("OTP sent for password reset", zap.String("email", req.Email))
	Response(c, http.StatusOK, "Jika email terdaftar, kode OTP telah dikirim ke email Anda", nil)
}

// ===================================================================
// ENDPOINT 2: POST /api/auth/verify-otp
// ===================================================================
// Memvalidasi apakah OTP cocok dan belum expired.
func (pc *PasswordResetController) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format data tidak valid. Pastikan email dan OTP 6 digit terisi.", nil)
		return
	}

	// 1. Cari record password_reset terbaru yang belum digunakan untuk email ini
	var resetRecord models.PasswordReset
	if err := pc.DB.
		Where("email = ? AND is_used = ?", req.Email, false).
		Order("created_at DESC").
		First(&resetRecord).Error; err != nil {
		Response(c, http.StatusBadRequest, "Kode OTP tidak valid atau sudah digunakan", nil)
		return
	}

	// 2. Cek apakah OTP sudah expired
	if time.Now().After(resetRecord.ExpiresAt) {
		// Tandai sebagai used agar tidak bisa dicoba lagi
		pc.DB.Model(&resetRecord).Update("is_used", true)
		Response(c, http.StatusBadRequest, "Kode OTP sudah kedaluwarsa. Silakan minta kode baru.", nil)
		return
	}

	// 3. Verifikasi OTP dengan hash yang tersimpan di database
	if !helper.CheckPasswordHash(req.OTP, resetRecord.OTPHash) {
		Response(c, http.StatusBadRequest, "Kode OTP tidak valid", nil)
		return
	}

	logger.Info("OTP verified successfully", zap.String("email", req.Email))
	Response(c, http.StatusOK, "Kode OTP valid. Silakan buat password baru.", nil)
}

// ===================================================================
// ENDPOINT 3: POST /api/auth/reset-password
// ===================================================================
// Menyimpan password baru setelah OTP terverifikasi.
// Double validation: memvalidasi ulang OTP + set password baru dalam satu transaksi.
func (pc *PasswordResetController) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format data tidak valid. Password minimal 6 karakter.", nil)
		return
	}

	// 1. Validasi password dan konfirmasi password harus cocok
	if req.NewPassword != req.ConfirmPassword {
		Response(c, http.StatusBadRequest, "Password baru dan konfirmasi password tidak cocok", nil)
		return
	}

	// 2. Gunakan DB Transaction untuk menjamin konsistensi data
	err := pc.DB.Transaction(func(tx *gorm.DB) error {
		// 2a. Cari record OTP terbaru yang belum digunakan
		var resetRecord models.PasswordReset
		if err := tx.
			Where("email = ? AND is_used = ?", req.Email, false).
			Order("created_at DESC").
			First(&resetRecord).Error; err != nil {
			return err
		}

		// 2b. Cek expired
		if time.Now().After(resetRecord.ExpiresAt) {
			tx.Model(&resetRecord).Update("is_used", true)
			return gorm.ErrRecordNotFound // Akan ditangkap di bawah
		}

		// 2c. Verifikasi ulang OTP (double validation demi keamanan)
		if !helper.CheckPasswordHash(req.OTP, resetRecord.OTPHash) {
			return gorm.ErrRecordNotFound
		}

		// 2d. Cari employee berdasarkan email
		var employee models.Employee
		if err := tx.Where("email = ?", req.Email).First(&employee).Error; err != nil {
			return err
		}

		// 2e. Cari user berdasarkan employee_id
		var user models.User
		if err := tx.Where("employee_id = ?", employee.ID).First(&user).Error; err != nil {
			return err
		}

		// 2f. Hash password baru
		newHash, err := helper.HashPassword(req.NewPassword)
		if err != nil {
			return err
		}

		// 2g. Update password di tabel users
		if err := tx.Model(&user).Update("password_hash", newHash).Error; err != nil {
			return err
		}

		// 2h. Tandai OTP sebagai sudah digunakan (invalidasi)
		if err := tx.Model(&resetRecord).Update("is_used", true).Error; err != nil {
			return err
		}

		// 2i. Invalidasi juga semua OTP lain untuk email ini
		tx.Model(&models.PasswordReset{}).
			Where("email = ? AND is_used = ? AND id != ?", req.Email, false, resetRecord.ID).
			Update("is_used", true)

		// 2j. Log activity
		helper.LogActivity(tx, user.ID, "RESET_PASSWORD", "Password berhasil direset via OTP", c.ClientIP())

		return nil
	})

	if err != nil {
		logger.Error("Failed to reset password", zap.String("email", req.Email), zap.Error(err))
		Response(c, http.StatusBadRequest, "Gagal mereset password. OTP tidak valid atau sudah kedaluwarsa.", nil)
		return
	}

	logger.Info("Password reset successfully", zap.String("email", req.Email))
	Response(c, http.StatusOK, "Password berhasil direset. Silakan login dengan password baru.", nil)
}
