package helper

import (
	"KPI_System_Backend/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// --- Password Hashing (Bcrypt) ---

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// --- JWT Generation & Validation ---

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID uint, username, role string) (string, error) {
	// Pastikan config sudah terload, jika 0 pakai default
	expiry := config.JWTExpiry
	if expiry == 0 {
		expiry = 24
	}
	expirationTime := time.Now().Add(time.Hour * time.Duration(expiry))

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	secret := config.JWTSecret
	if secret == "" {
		secret = "rahasia_super_aman_cakra_123"
	}
	jwtKey := []byte(secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateJWT(tokenString string) (*Claims, error) {
	secret := config.JWTSecret
	if secret == "" {
		secret = "rahasia_super_aman_cakra_123"
	}
	jwtKey := []byte(secret)

	claims := &Claims{}

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

	return claims, nil
}