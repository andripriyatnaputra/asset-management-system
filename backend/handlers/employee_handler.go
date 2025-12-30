// File: backend/handlers/employee_handler.go
package handlers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// 🔐 ROLE DEFINITIONS
// ============================================================
type RoleType string

const (
	RoleSuperAdmin RoleType = "super_admin"
	RoleAdmin      RoleType = "admin"
	RoleManager    RoleType = "manager"
	RoleAuditor    RoleType = "auditor"
	RoleEmployee   RoleType = "employee"
)

// DTO yang dipakai frontend (lihat EmployeeFormModal.tsx)
type EmployeeDTO struct {
	ID           int64    `json:"id,omitempty"`
	EmployeeNIK  string   `json:"employee_nik"`
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	DepartmentID *int64   `json:"department_id"` // nullable
	Role         RoleType `json:"role"`
}

// ============================================================
// 🧮 UTILITY
// ============================================================
func hash(pass string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(b), err
}

func generateStrongPassword(length int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	digits := []rune("0123456789")
	symbols := []rune("!@#$%^&*()-_=+[]{};:,.<>?")
	if length < 10 {
		length = 10
	}
	rand.Seed(time.Now().UnixNano())
	res := []rune{
		letters[rand.Intn(len(letters))],
		unicode.ToUpper(letters[rand.Intn(len(letters))]),
		digits[rand.Intn(len(digits))],
		symbols[rand.Intn(len(symbols))],
	}
	all := append(append(letters, digits...), symbols...)
	for len(res) < length {
		res = append(res, all[rand.Intn(len(all))])
	}
	rand.Shuffle(len(res), func(i, j int) { res[i], res[j] = res[j], res[i] })
	return string(res)
}

// -------- Handlers --------

// ============================================================
// 👥 LIST EMPLOYEES (with delegation, score & safe join)
// ============================================================
func ListEmployees(c *gin.Context) {
	// 🔹 Query parameter dari frontend
	search := c.Query("q")
	sortBy := c.DefaultQuery("sort_by", "e.name")
	sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "ASC"))
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	// 🔹 Validasi dasar
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "ASC"
	}

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 🔹 Base query (tanpa filter)
	baseQuery := `
		FROM employees e
		LEFT JOIN departments d ON e.department_id = d.id
		LEFT JOIN asset_assignments aa ON aa.employee_id = e.id
		WHERE e.deleted_at IS NULL
	`

	// 🔹 Tambahkan pencarian
	whereClause := ""
	if search != "" {
		search = strings.TrimSpace(search)
		whereClause = fmt.Sprintf(
			" AND (LOWER(e.name) LIKE LOWER('%%%s%%') OR LOWER(e.email) LIKE LOWER('%%%s%%') OR LOWER(e.employee_nik) LIKE LOWER('%%%s%%'))",
			search, search, search,
		)
	}

	// 🔹 Query utama (data)
	query := fmt.Sprintf(`
		SELECT 
			e.id,
			e.employee_nik,
			e.name,
			e.email,
			e.role,
			e.department_id,
			COALESCE(d.name, '-') AS department_name,
			e.last_login_at,
			e.created_at,
			e.updated_at,
			COUNT(aa.id) FILTER (WHERE aa.returned_at IS NULL) AS active_assets,
			COALESCE((
				SELECT COUNT(*) 
				FROM role_delegations r 
				WHERE r.delegatee_id = e.id 
				  AND NOW() BETWEEN r.start_date AND r.end_date
			), 0) AS delegation_count
		%s
		%s
		GROUP BY e.id, e.employee_nik, e.name, e.email, e.role, e.department_id, d.name, e.last_login_at, e.created_at, e.updated_at
		ORDER BY %s %s
		LIMIT %d OFFSET %d
	`, baseQuery, whereClause, sortBy, sortDir, limit, offset)

	// 🔹 Query total record
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT e.id) %s %s", baseQuery, whereClause)

	// 🔹 Eksekusi query total
	var totalRecords int
	if err := database.Pool.QueryRow(c.Request.Context(), countQuery).Scan(&totalRecords); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count employees"})
		return
	}

	// 🔹 Eksekusi query utama
	rows, err := database.Pool.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query employees"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID              int64      `json:"id"`
		NIK             string     `json:"employee_nik"`
		Name            string     `json:"name"`
		Email           string     `json:"email"`
		Role            string     `json:"role"`
		DepartmentName  string     `json:"department_name"`
		LastLoginAt     *time.Time `json:"last_login_at"`
		CreatedAt       *time.Time `json:"created_at"`
		UpdatedAt       *time.Time `json:"updated_at"`
		ActiveAssets    int        `json:"active_assets"`
		DelegationCount int        `json:"delegation_count"`
		HealthScore     float64    `json:"employee_health_score"`
	}

	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.ID,
			&r.NIK,
			&r.Name,
			&r.Email,
			&r.Role,
			new(any), // skip department_id
			&r.DepartmentName,
			&r.LastLoginAt,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.ActiveAssets,
			&r.DelegationCount,
		); err != nil {
			log.Printf("[WARN] scan employee: %v", err)
			continue
		}

		// 🧮 Hitung health score
		days := 0.0
		if r.LastLoginAt != nil {
			days = time.Since(*r.LastLoginAt).Hours() / 24
		}
		score := 100.0 - math.Min(days, 100)
		if r.ActiveAssets > 0 {
			score += 5
		}
		if r.DelegationCount > 0 {
			score += 5
		}
		if score > 100 {
			score = 100
		}
		r.HealthScore = score

		// 🚨 Kirim notifikasi jika tidak aktif > 90 hari
		if days > 90 {
			msg := fmt.Sprintf("Akun %s (%s) tidak aktif >90 hari", r.Name, r.Email)
			services.BroadcastAlert(msg, "warning")
		}

		list = append(list, r)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[WARN] employee iteration error: %v", err)
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(limit)))
	c.JSON(http.StatusOK, gin.H{
		"data": list,
		"pagination": gin.H{
			"page":          page,
			"limit":         limit,
			"total_pages":   totalPages,
			"total_records": totalRecords,
		},
	})
}

// ============================================================
// 🧾 CREATE EMPLOYEE (Governed)
// ============================================================
func CreateEmployee(c *gin.Context) {
	var body struct {
		Name         string   `json:"name" binding:"required"`
		Email        string   `json:"email" binding:"required,email"`
		DepartmentID *int64   `json:"department_id"`
		Role         RoleType `json:"role"`
		Password     *string  `json:"password,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pass := generateStrongPassword(12)
	if body.Password != nil && *body.Password != "" {
		pass = *body.Password
	}
	hashPass, _ := hash(pass)
	nik := fmt.Sprintf("EMP-%d", time.Now().UnixNano()%1000000)

	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM employees WHERE email=$1 AND deleted_at IS NULL)`, body.Email).Scan(&exists)
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO employees (employee_nik,name,email,department_id,role,password_hash)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id`,
		nik, body.Name, body.Email, body.DepartmentID, body.Role, hashPass,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	middleware.LogAction(c, "employees", id, "CREATE", body)
	c.JSON(http.StatusCreated, gin.H{"id": id, "generated_password": pass})
}

// ============================================================
// ✏️ UPDATE EMPLOYEE (+ delegation sync)
// ============================================================
func UpdateEmployee(c *gin.Context) {
	id := mustAtoi64(c.Param("id"))
	var body struct {
		Name         *string   `json:"name"`
		Email        *string   `json:"email"`
		Role         *RoleType `json:"role"`
		DepartmentID *int64    `json:"department_id"`
		Password     *string   `json:"password"` // NEW
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password if provided
	var passwordHash *string
	if body.Password != nil && *body.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		hashStr := string(hashed)
		passwordHash = &hashStr
	}

	_, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE employees SET 
           name           = COALESCE($1,name),
           email          = COALESCE($2,email),
           role           = COALESCE($3,role),
           department_id  = COALESCE($4,department_id),
           password_hash  = COALESCE($5,password_hash),
           updated_at     = NOW()
         WHERE id = $6`,
		body.Name, body.Email, body.Role, body.DepartmentID, passwordHash, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "employees", id, "UPDATE", body)

	c.JSON(http.StatusOK, gin.H{"message": "employee updated"})
}

// ============================================================
// 🗑 DELETE EMPLOYEE (soft delete + alert)
// ============================================================
func DeleteEmployee(c *gin.Context) {
	id := mustAtoi64(c.Param("id"))

	// 🔹 Cek apakah masih memiliki asset aktif
	var activeAssets int
	err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM asset_assignments WHERE employee_id=$1 AND returned_at IS NULL`, id).
		Scan(&activeAssets)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check asset linkage"})
		return
	}
	if activeAssets > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active asset assignments exist"})
		return
	}

	// 🔹 Cek apakah masih punya delegasi aktif
	var delegations int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM role_delegations WHERE delegatee_id=$1 AND NOW() BETWEEN start_date AND end_date`, id).
		Scan(&delegations)
	if delegations > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active role delegations exist"})
		return
	}

	// 🔹 Soft delete jika aman
	_, err = database.Pool.Exec(c.Request.Context(),
		`UPDATE employees SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	services.BroadcastAlert(fmt.Sprintf("Employee ID %d deleted", id), "warning")
	middleware.LogAction(c, "employees", id, "DELETE", nil)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ============================================================
// 📥 IMPORT EMPLOYEES (role default employee + generic password)
// ============================================================
func ImportEmployeesFromCSV(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}

	f, _ := fh.Open()
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1

	first := true
	var created, updated, failed int

	// Password generic (ganti sesuai kebutuhan Anda)
	genericPw := "Welcome123!"
	genericPwHash, _ := hash(genericPw)

	for {
		rec, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			failed++
			continue
		}

		// Skip header
		if first {
			first = false
			continue
		}

		// CSV: NO (0) | NAMA (1) | NIK (2) | EMAIL (3) | DEPT (4)
		if len(rec) < 4 {
			failed++
			continue
		}

		name := strings.TrimSpace(rec[1])
		nik := strings.TrimSpace(rec[2])
		email := strings.TrimSpace(rec[3])

		var deptName string
		if len(rec) >= 5 {
			deptName = strings.TrimSpace(rec[4])
		}

		if name == "" || email == "" || !strings.Contains(email, "@") {
			failed++
			continue
		}

		// Default role
		role := RoleEmployee

		// Optional: auto-create department
		var deptID *int64
		if deptName != "" {
			err = database.Pool.QueryRow(c.Request.Context(),
				`INSERT INTO departments(name)
				 VALUES($1)
				 ON CONFLICT(name) DO UPDATE SET name=EXCLUDED.name
				 RETURNING id`,
				deptName,
			).Scan(&deptID)

			if err != nil {
				deptID = nil
			}
		}

		// Insert or update
		var id int64
		err = database.Pool.QueryRow(c.Request.Context(), `
			INSERT INTO employees (employee_nik, name, email, role, department_id, password_hash)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT(email) DO UPDATE SET
				name = EXCLUDED.name,
				role = EXCLUDED.role,
				employee_nik = EXCLUDED.employee_nik,
				department_id = EXCLUDED.department_id
			RETURNING id`,
			nik,
			name,
			email,
			role,
			deptID,
			genericPwHash, // 🔐 always generic password
		).Scan(&id)

		if err != nil {
			failed++
			continue
		}

		// Determine whether created or updated
		var existedBefore bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM employees WHERE email=$1 AND id <> $2)`,
			email, id,
		).Scan(&existedBefore)

		if existedBefore {
			updated++
		} else {
			created++
		}
	}

	msg := fmt.Sprintf("Import selesai: created=%d, updated=%d, failed=%d", created, updated, failed)
	services.BroadcastAlert(msg, "info")

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// GET /employees/me/assets
func GetMyAssignedAssets(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok || uid == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var userID int64
	switch v := uid.(type) {
	case int:
		userID = int64(v)
	case int64:
		userID = v
	case float64:
		userID = int64(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id type"})
		return
	}

	type Row struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		AssetTag      string `json:"asset_tag"`
		AssetTypeName string `json:"asset_type_name"`
	}
	rows, err := database.Pool.Query(c, `
		SELECT a.id, a.name, a.asset_tag, at.name AS asset_type_name
		  FROM assets a
		  JOIN asset_assignments aa ON aa.asset_id = a.id AND aa.returned_at IS NULL
		  JOIN asset_types at ON at.id = a.asset_type_id
		 WHERE aa.employee_id = $1
		 ORDER BY a.name`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assets"})
		return
	}
	defer rows.Close()

	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Name, &r.AssetTag, &r.AssetTypeName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

// GET /employees/:id
func GetEmployee(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var (
		empID       int64
		employeeNIK string
		name        string
		email       string
		deptID      *int64
		deptName    *string
		role        RoleType
	)
	err = database.Pool.QueryRow(c, `
		SELECT e.id, e.employee_nik, e.name, e.email, e.department_id, d.name, e.role
		  FROM employees e
		  LEFT JOIN departments d ON d.id = e.department_id
		 WHERE e.id = $1 AND e.deleted_at IS NULL
	`, id).Scan(&empID, &employeeNIK, &name, &email, &deptID, &deptName, &role)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch employee"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              empID,
		"employee_nik":    employeeNIK,
		"name":            name,
		"email":           email,
		"department_id":   deptID,
		"department_name": deptName,
		"role":            role,
	})
}

// ============================================================
// 🔁 RESET PASSWORD (secure + rotation alert)
// ============================================================
func ResetEmployeePassword(c *gin.Context) {
	id := mustAtoi64(c.Param("id"))
	newPass := generateStrongPassword(12)
	hashPass, _ := hash(newPass)
	_, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE employees SET password_hash=$1,updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`,
		hashPass, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	msg := fmt.Sprintf("Password reset for Employee #%d", id)
	services.BroadcastAlert(msg, "warning")
	middleware.LogAction(c, "employees", id, "RESET_PASSWORD", gin.H{"new_password": newPass})
	c.JSON(http.StatusOK, gin.H{"message": msg, "generated_password": newPass})
}

// GET /employees/me
// ============================================================
// 👤 MY PROFILE
// ============================================================
func GetMyProfile(c *gin.Context) {
	uid, _ := c.Get("user_id")
	if uid == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var userID int64
	switch v := uid.(type) {
	case int64:
		userID = v
	case int:
		userID = int64(v)
	case float64:
		userID = int64(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id type"})
		return
	}

	var name, email, role, dept string
	var lastLogin *time.Time
	err := database.Pool.QueryRow(c.Request.Context(), `
		SELECT e.name,e.email,e.role,COALESCE(d.name,''),e.last_login_at
		  FROM employees e LEFT JOIN departments d ON e.department_id=d.id
		 WHERE e.id=$1 AND e.deleted_at IS NULL`, userID).
		Scan(&name, &email, &role, &dept, &lastLogin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "profile fetch failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"name": name, "email": email, "role": role,
		"department": dept, "last_login_at": lastLogin,
	})
}

// PUT /employees/me
func UpdateMyProfile(c *gin.Context) {
	uid, _ := c.Get("user_id")
	if uid == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var userID int64
	switch v := uid.(type) {
	case int64:
		userID = v
	case int:
		userID = int64(v)
	case float64:
		userID = int64(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id type"})
		return
	}

	var req struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	_, err := database.Pool.Exec(c, `
        UPDATE employees
           SET name=$1, email=$2, updated_at=NOW()
         WHERE id=$3 AND deleted_at IS NULL
    `, req.Name, req.Email, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}
	middleware.LogAction(c, "employees", userID, "UPDATE_SELF", req)
	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}
