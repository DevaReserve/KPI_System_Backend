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

// AuthMiddleware: Memvalidasi Token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Cek query param juga (opsional, kadang berguna untuk download file)
			authHeader = c.Query("token")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, global_var.ResponseFormat{
					Status:  http.StatusUnauthorized,
					Message: "Authorization header required",
					Data:    nil,
				})
				c.Abort()
				return
			}
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		claims, err := helper.ValidateJWT(tokenString)
		if err != nil {
			logger.Warn("Invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, global_var.ResponseFormat{
				Status:  http.StatusUnauthorized,
				Message: "Invalid token or expired",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// PENTING: Set context keys yang konsisten
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		// Gunakan "role" (bukan userRole) agar sesuai dengan routes.go logic
		c.Set("role", claims.Role) 

		c.Next()
	}
}

// CORSMiddleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RoleCheckMiddleware: Mendukung MULTI ROLE (Variadic ...string)
func RoleCheckMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil "role" yang diset di AuthMiddleware
		userRoleInterface, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, global_var.ResponseFormat{
				Status:  http.StatusUnauthorized,
				Message: "Unauthorized (Role not found)",
				Data:    nil,
			})
			c.Abort()
			return
		}

		userRole := userRoleInterface.(string)
		isAllowed := false

		// Cek apakah role user ada di daftar allowedRoles
		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, global_var.ResponseFormat{
				Status:  http.StatusForbidden,
				Message: "Akses Ditolak: Anda tidak memiliki izin untuk fitur ini",
				Data:    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}