// File: backend/handlers/auth_handler_test.go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// TestMain akan berjalan sekali sebelum semua tes di paket ini dijalankan.
func TestMain(m *testing.M) {
	// Muat .env untuk mendapatkan JWT_SECRET_KEY
	if os.Getenv("CI") == "" {
		if err := godotenv.Load("../../.env"); err != nil {
			log.Fatalf("Error loading .env file for local testing: %v", err)
		}
	}

	// Atur environment variable untuk koneksi ke database tes
	os.Setenv("DATABASE_URL", "postgres://admin_test:secret_test@localhost:5433/asset_db_test?sslmode=disable")
	database.Connect()

	_, err := database.Pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS cdc CASCADE; DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	if err != nil {
		log.Fatalf("Could not clean test database: %v", err)
	}
	log.Println("Test database cleaned successfully.")

	// --- PERBAIKAN DI SINI: Jalankan init.sql di database tes ---
	sqlBytes, err := os.ReadFile("../../database/init/init.sql")
	if err != nil {
		log.Fatalf("Could not read init.sql file: %v", err)
	}

	// init.sql hasil dump production memuat perintah ownership/grant yang bisa gagal
	// di lingkungan tes lokal (contoh role "admin" tidak ada). Untuk tes handler,
	// schema dan data awal tetap dibutuhkan, tapi ownership statement aman di-skip.
	cleanedSQL := sanitizeInitSQLForTests(string(sqlBytes))
	_, err = database.Pool.Exec(context.Background(), cleanedSQL)
	if err != nil {
		log.Fatalf("Could not execute init.sql on test database: %v", err)
	}
	// -----------------------------------------------------------

	// Jalankan tes
	exitCode := m.Run()

	os.Exit(exitCode)
}

func sanitizeInitSQLForTests(sql string) string {
	var cleanedLines []string
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		skipLine := strings.Contains(trimmed, " OWNER TO ") ||
			strings.HasPrefix(trimmed, "GRANT ") ||
			strings.HasPrefix(trimmed, "REVOKE ") ||
			strings.HasPrefix(trimmed, "ALTER DEFAULT PRIVILEGES") ||
			strings.HasPrefix(trimmed, "SET SESSION AUTHORIZATION") ||
			strings.Contains(trimmed, "set_config('search_path', '', false)")

		if skipLine {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	return strings.Join(cleanedLines, "\n")
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/auth/login", Login)
	return r
}

func TestLoginHandler(t *testing.T) {
	r := setupRouter()

	// Setup: Hapus semua data dari tabel (kecuali data awal dari init.sql)
	// Kita gunakan TRUNCATE untuk reset cepat
	database.Pool.Exec(context.Background(), "TRUNCATE TABLE employees, departments RESTART IDENTITY CASCADE")
	_, err := database.Pool.Exec(context.Background(), "INSERT INTO departments (id, name) VALUES (1, 'IT Test') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name")
	assert.NoError(t, err)

	// Setup: Buat user tes tambahan
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err = database.Pool.Exec(context.Background(),
		`INSERT INTO public.employees (employee_nik, name, email, password_hash, role, department_id) 
		 VALUES ('TEST-001', 'Test User', 'test@example.com', $1, 'employee', 1)`,
		string(hashedPassword))
	assert.NoError(t, err)

	// Skenario 1: Login Sukses
	t.Run("Successful Login", func(t *testing.T) {
		// ... (sisa kode tes ini tidak berubah)
		loginPayload := map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}
		payloadBytes, _ := json.Marshal(loginPayload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(t, response, "token")
	})

	// Skenario 2: Login Gagal (Password Salah)
	t.Run("Failed Login - Wrong Password", func(t *testing.T) {
		// ... (sisa kode tes ini tidak berubah)
		loginPayload := map[string]string{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}
		payloadBytes, _ := json.Marshal(loginPayload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
