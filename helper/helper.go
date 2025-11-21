package helper

import (
	"KPI_System_Backend/config" // Import config untuk JWT
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4" // Import untuk JWT
	"golang.org/x/crypto/bcrypt"   // Import untuk Password Hashing
)

// --- Password Hashing (Bcrypt) ---

// HashPassword digunakan saat membuat user baru
func HashPassword(password string) (string, error) {
	// Cost 10 adalah standar yang baik
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

// CheckPasswordHash digunakan saat login
func CheckPasswordHash(password, hash string) bool {
	// Membandingkan password (string) dengan hash (dari DB)
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil // Jika err == nil, password cocok
}

// --- JWT Generation & Validation ---

// Claims struct adalah data yang kita simpan di dalam token JWT
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT membuat token baru untuk user
func GenerateJWT(userID uint, username, role string) (string, error) {
	// Ambil durasi token dari config (misal: 72 jam)
	expirationTime := time.Now().Add(time.Hour * time.Duration(config.JWTExpiry))

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			// Set waktu kedaluwarsa
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Ambil Kunci Rahasia JWT dari config
	jwtKey := []byte(config.JWTSecret)

	// Buat token baru dengan metode HS256 dan claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Tandatangani token dengan kunci rahasia
	tokenString, err := token.SignedString(jwtKey)

	return tokenString, err
}

// ValidateJWT memverifikasi token string
func ValidateJWT(tokenString string) (*Claims, error) {
	jwtKey := []byte(config.JWTSecret)

	claims := &Claims{}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, errors.New("invalid token signature")
		}
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Token valid, kembalikan claims
	return claims, nil
}