package middleware

import (
	"KPI_System_Backend/global_var"
	"KPI_System_Backend/helper"
	"KPI_System_Backend/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware dari file Anda sebelumnya
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Missing authorization header")
			c.JSON(http.StatusUnauthorized, global_var.ResponseFormat{
				Status:  http.StatusUnauthorized,
				Message: "Authorization header required",
				Data:    nil,
			})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		claims, err := helper.ValidateJWT(tokenString)
		if err != nil {
			logger.Warn("Invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, global_var.ResponseFormat{
				Status:  http.StatusUnauthorized,
				Message: "Invalid token",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("userRole", claims.Role)

		c.Next()
	}
}

// CORSMiddleware dari file Anda sebelumnya
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RoleCheckMiddleware yang baru saja kita buat
func RoleCheckMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil role dari context, yang sudah di-set oleh AuthMiddleware
		userRole, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusForbidden, global_var.ResponseFormat{
				Status:  http.StatusForbidden,
				Message: "Akses ditolak (role tidak ditemukan)",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// Cek apakah role pengguna ada di dalam daftar yang diizinkan
		isAllowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, global_var.ResponseFormat{
				Status:  http.StatusForbidden,
				Message: "Anda tidak memiliki hak akses untuk fitur ini",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// Jika diizinkan, lanjutkan
		c.Next()
	}
}