package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	auth "github.com/andripriyatnaputra/asset-management-system/backend/auth"
	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	middleware "github.com/andripriyatnaputra/asset-management-system/backend/security"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ============================================================
// 🔐 AUTHENTICATE (Fix SET LOCAL error + ISO A++ Context)
// ============================================================
func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		if middleware.IsTokenRevoked(tokenString) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token_revoked"})
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return auth.GetSecret(), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// ✅ Ambil data dari JWT
		userID := int64(claims["user_id"].(float64))
		role := strings.ToLower(claims["role"].(string))
		var deptID *int64
		if v, ok := claims["department_id"]; ok {
			tmp := int64(v.(float64))
			deptID = &tmp
		}

		// ✅ Simpan di context agar handler bisa pakai
		c.Set("user_id", userID)
		fmt.Println("[DEBUG] set user_id =", userID)
		c.Set("role", role)
		if deptID != nil {
			c.Set("department_id", *deptID)
		}

		// =======================================================
		// 🔧 Set context variable di PostgreSQL (RLS / Audit)
		//    * Gunakan fmt.Sprintf karena SET LOCAL tidak terima $1
		// =======================================================
		ctx := context.Background()

		// app.current_user_id
		sqlUser := fmt.Sprintf("SET app.current_user_id = %d", userID)
		if _, err := database.Pool.Exec(ctx, sqlUser); err != nil {
			// tidak fatal, tapi dicatat ke log
			fmt.Printf("[AUTH][WARN] gagal set user context: %v\n", err)
		}

		// app.current_department
		if deptID != nil && *deptID > 0 {
			sqlDept := fmt.Sprintf("SET app.current_department = '%d'", *deptID)
			database.Pool.Exec(ctx, sqlDept)
		} else {
			// Gunakan SET (bukan SET LOCAL) dan 'NULL' literal
			database.Pool.Exec(ctx, "SET app.current_department = 'NULL'")
		}

		// lanjutkan ke handler berikutnya
		c.Next()
	}
}

// ============================================================
// 🔑 AUTHORIZE (Case Insensitive + ISO A++ Compatible)
// ============================================================
func Authorize(allowedRoles ...string) gin.HandlerFunc {
	// Normalisasi daftar allowed roles → lowercase
	allowed := map[string]struct{}{}
	for _, r := range allowedRoles {
		allowed[strings.ToLower(r)] = struct{}{}
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unauthorized: no role in context"})
			return
		}

		// Normalisasi role user
		userRole := strings.ToLower(fmt.Sprintf("%v", roleVal))

		// super_admin override (akses penuh)
		if userRole == "super_admin" {
			c.Next()
			return
		}

		// Cocokkan role (case-insensitive)
		if _, ok := allowed[userRole]; ok {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": fmt.Sprintf("access denied for role %s", userRole),
		})
	}
}

// ============================================================
// 🧭 AUTHORIZE DEPARTMENT
// ============================================================
func AuthorizeDepartment(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userDept, ok := c.Get("department_id")
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no department in context"})
			return
		}

		reqDept := c.Param(param)
		if reqDept == "" {
			c.Next()
			return
		}

		if fmt.Sprintf("%v", userDept) != reqDept {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unauthorized department access"})
			return
		}

		c.Next()
	}
}
