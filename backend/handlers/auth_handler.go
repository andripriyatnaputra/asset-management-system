// File: backend/handlers/auth_handler.go
package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/auth" // <- Impor package auth kita
	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

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

	// Cari user berdasarkan email
	log.Println("Mencari pengguna dengan email:", req.Email)
	query := "SELECT id, email, password_hash, role, name FROM employees WHERE email = $1"
	err := database.Pool.QueryRow(context.Background(), query, req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Name)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Bandingkan password yang diberikan dengan hash di database
	log.Println("Pengguna ditemukan, membandingkan password...")
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Jika berhasil, buat token JWT
	log.Println("Login berhasil, membuat token...")
	tokenString, err := auth.GenerateToken(user.ID, user.Email, user.Role, user.Name)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword menangani logika perubahan password untuk pengguna yang sedang login
func ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 1. Ambil userID dari context yang sudah di-set oleh middleware Authenticate
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	// 2. Ambil hash password saat ini dari database
	var currentPasswordHash string
	err := database.Pool.QueryRow(context.Background(), "SELECT password_hash FROM employees WHERE id = $1", userID).Scan(&currentPasswordHash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 3. Verifikasi password lama
	err = bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(req.OldPassword))
	if err != nil {
		// Jika tidak cocok, kembalikan error
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password lama salah."})
		return
	}

	// 4. Hash password baru
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	// 5. Update password di database
	_, err = database.Pool.Exec(context.Background(), "UPDATE employees SET password_hash = $1 WHERE id = $2", string(newHashedPassword), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diperbarui."})
}
