// File: backend/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/andripriyatnaputra/asset-management-system/backend/auth"
	"github.com/gin-gonic/gin"
)

// Authenticate adalah middleware untuk memeriksa validitas token JWT
func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			return
		}

		// Gunakan fungsi validasi terpusat yang sudah kita buat
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Simpan informasi user di dalam context
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("userEmail", claims.Email)
		c.Set("userName", claims.UserName)

		// Lanjutkan ke handler berikutnya
		c.Next()
	}
}

// Authorize adalah middleware untuk memeriksa peran pengguna
func Authorize(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil peran pengguna dari context yang sudah di-set oleh middleware Authenticate
		userRole, exists := c.Get("userRole")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
			return
		}

		// Super admin boleh mengakses semuanya
		if userRole.(string) == "super_admin" {
			c.Next()
			return
		}

		if userRole.(string) != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You are not authorized to perform this action"})
			return
		}

		c.Next()
	}
}
