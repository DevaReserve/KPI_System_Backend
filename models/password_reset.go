package models

import (
	"time"
)

// PasswordReset menyimpan data OTP untuk fitur reset password.
// OTPHash menyimpan OTP yang sudah di-hash bcrypt (bukan plaintext) untuk keamanan.
type PasswordReset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"size:100;index" json:"email"`
	OTPHash   string    `gorm:"size:255" json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	IsUsed    bool      `gorm:"default:false" json:"is_used"`
	CreatedAt time.Time `json:"created_at"`
}
