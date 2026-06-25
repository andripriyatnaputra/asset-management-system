package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/auth"
	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// 🔐 LOGIN HANDLER
// ============================================================
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	var user models.Employee

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	normalizedEmail := strings.TrimSpace(strings.ToLower(req.Email))
	fmt.Println("LOGIN ATTEMPT EMAIL =", normalizedEmail)

	query := `
			SELECT id, email, password_hash, role, name, department_id
			FROM public.employees 
			WHERE email = $1 AND deleted_at IS NULL
		`
	err := database.Pool.QueryRow(context.Background(), query, normalizedEmail).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Name, &user.DepartmentID,
	)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	user.Role = strings.ToLower(user.Role)

	fmt.Println("ROLE FROM DB:", user.Role)

	// 🔐 Password validation
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		services.SendSIEMAlert("Login gagal", fmt.Sprintf("Percobaan login gagal untuk %s", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 🪪 Generate JWT token pair (access + refresh)
	var deptID int64
	if user.DepartmentID != nil {
		deptID = *user.DepartmentID
	}
	accessToken, refreshToken, err := auth.GenerateTokenPair(user.ID, user.Email, user.Role, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token pair"})
		return
	}

	// 🕓 Async update last login
	go func() {
		_, _ = database.Pool.Exec(context.Background(),
			`UPDATE public.employees SET last_login_at = $1 WHERE id = $2`,
			time.Now(), user.ID)
	}()

	// 🧾 Audit log
	middleware.LogAction(c, "employees", user.ID, "LOGIN", gin.H{
		"email": user.Email,
		"role":  user.Role,
	})

	log.Printf("[LOGIN] user=%s (role=%s) berhasil login", user.Email, user.Role)

	// ✅ Single clean JSON response
	c.JSON(http.StatusOK, gin.H{
		"token":         accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":            user.ID,
			"name":          user.Name,
			"email":         user.Email,
			"role":          user.Role,
			"department_id": user.DepartmentID,
		},
	})
}

// ============================================================
// 🔁 REFRESH TOKEN HANDLER
// ============================================================
func RefreshToken(c *gin.Context) {
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	claims := &auth.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return auth.GetSecret(), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	newAccess, err := auth.GenerateToken(
		claims.UserID, claims.Email, claims.Role, claims.DepartmentID, 15*time.Minute,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh token"})
		return
	}

	middleware.LogAction(c, "employees", claims.UserID, "REFRESH_TOKEN", gin.H{"email": claims.Email})
	c.JSON(http.StatusOK, gin.H{"token": newAccess})
}

// ============================================================
// 🔑 CHANGE PASSWORD HANDLER
// ============================================================
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	userVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var userID int64
	switch v := userVal.(type) {
	case int64:
		userID = v
	case int:
		userID = int64(v)
	case float64:
		userID = int64(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
		return
	}

	var currentPasswordHash string
	err := database.Pool.QueryRow(context.Background(),
		`SELECT password_hash FROM public.employees WHERE id = $1 AND deleted_at IS NULL`,
		userID).Scan(&currentPasswordHash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password lama salah"})
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	_, err = database.Pool.Exec(context.Background(),
		`UPDATE public.employees SET password_hash = $1 WHERE id = $2`, newHash, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	middleware.LogAction(c, "employees", userID, "CHANGE_PASSWORD", nil)
	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diperbarui."})
}
