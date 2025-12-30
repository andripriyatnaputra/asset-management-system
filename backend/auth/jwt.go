// File: backend/auth/jwt.go
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ===============================
// 📦 STRUCT CLAIMS
// ===============================
type Claims struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	DepartmentID int64  `json:"department_id"`
	jwt.RegisteredClaims
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET_KEY"))

// Atau default fallback:
func GetSecret() []byte {
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("jwtsecretpassword") // fallback kalau env kosong
	}
	return jwtSecret
}

// ===============================
// 🔑 GenerateToken
// ===============================
func GenerateToken(userID int64, email, role string, departmentID int64, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID:       userID,
		Email:        email,
		Role:         role,
		DepartmentID: departmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetSecret())
}

// ===============================
// 🧩 GenerateTokenPair
// ===============================
func GenerateTokenPair(userID int64, email, role string, departmentID int64) (access, refresh string, err error) {
	access, err = GenerateToken(userID, email, role, departmentID, 15*time.Minute)
	if err != nil {
		return "", "", err
	}
	refresh, err = GenerateToken(userID, email, role, departmentID, 7*24*time.Hour)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// ValidateToken memvalidasi token string dan mengembalikan claims jika valid
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return GetSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
