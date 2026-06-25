package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/security"
	"github.com/gin-gonic/gin"
)

// GetLDAPConfigs mengembalikan semua konfigurasi LDAP/AD (password disembunyikan).
func GetLDAPConfigs(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT id, name, host, port, use_tls, base_dn, bind_dn,
		       user_filter, field_map, is_active, created_at, updated_at
		FROM ldap_sync_configs ORDER BY name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.LDAPSyncConfig
	for rows.Next() {
		var l models.LDAPSyncConfig
		if err := rows.Scan(&l.ID, &l.Name, &l.Host, &l.Port, &l.UseTLS,
			&l.BaseDN, &l.BindDN, &l.UserFilter, &l.FieldMap,
			&l.IsActive, &l.CreatedAt, &l.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, l)
	}
	if list == nil {
		list = []models.LDAPSyncConfig{}
	}
	c.JSON(http.StatusOK, list)
}

// CreateLDAPConfig membuat konfigurasi LDAP baru.
func CreateLDAPConfig(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required"`
		Host         string  `json:"host" binding:"required"`
		Port         int     `json:"port"`
		UseTLS       bool    `json:"use_tls"`
		BaseDN       string  `json:"base_dn" binding:"required"`
		BindDN       string  `json:"bind_dn" binding:"required"`
		BindPassword *string `json:"bind_password"`
		UserFilter   string  `json:"user_filter"`
		FieldMap     string  `json:"field_map"` // JSON: {"sAMAccountName":"username"}
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Port == 0 {
		req.Port = 389
	}
	if req.UserFilter == "" {
		req.UserFilter = "(objectClass=person)"
	}
	if req.FieldMap == "" {
		req.FieldMap = `{"sAMAccountName":"username","cn":"name","mail":"email"}`
	}

	// Encrypt bind_password before storing
	var encryptedPassword *string
	if req.BindPassword != nil && *req.BindPassword != "" {
		enc, err := security.EncryptLDAP(*req.BindPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengenkripsi password: " + err.Error()})
			return
		}
		encryptedPassword = &enc
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO ldap_sync_configs (name, host, port, use_tls, base_dn, bind_dn, bind_password, user_filter, field_map)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id
	`, req.Name, req.Host, req.Port, req.UseTLS, req.BaseDN, req.BindDN,
		encryptedPassword, req.UserFilter, req.FieldMap).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "konfigurasi LDAP berhasil dibuat"})
}

// UpdateLDAPConfig mengupdate konfigurasi LDAP yang ada.
func UpdateLDAPConfig(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Host         *string `json:"host"`
		Port         *int    `json:"port"`
		UseTLS       *bool   `json:"use_tls"`
		BaseDN       *string `json:"base_dn"`
		BindDN       *string `json:"bind_dn"`
		BindPassword *string `json:"bind_password"`
		UserFilter   *string `json:"user_filter"`
		FieldMap     *string `json:"field_map"`
		IsActive     *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Encrypt bind_password if a new one was provided
	var encPwd *string
	if req.BindPassword != nil && *req.BindPassword != "" {
		enc, err := security.EncryptLDAP(*req.BindPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengenkripsi password: " + err.Error()})
			return
		}
		encPwd = &enc
	}

	_, err := database.Pool.Exec(c, `
		UPDATE ldap_sync_configs SET
			host          = COALESCE($1, host),
			port          = COALESCE($2, port),
			use_tls       = COALESCE($3, use_tls),
			base_dn       = COALESCE($4, base_dn),
			bind_dn       = COALESCE($5, bind_dn),
			bind_password = COALESCE($6, bind_password),
			user_filter   = COALESCE($7, user_filter),
			field_map     = COALESCE($8, field_map),
			is_active     = COALESCE($9, is_active),
			updated_at    = now()
		WHERE id = $10
	`, req.Host, req.Port, req.UseTLS, req.BaseDN, req.BindDN,
		encPwd, req.UserFilter, req.FieldMap, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "konfigurasi LDAP berhasil diupdate"})
}

// TriggerLDAPSync memulai sinkronisasi LDAP (stub — log entry dibuat, proses actual via agent/cron).
func TriggerLDAPSync(c *gin.Context) {
	configID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	// Ambil config — bind_password di-decrypt sebelum dipakai koneksi
	var isActive bool
	var encryptedPwd *string
	err := database.Pool.QueryRow(c,
		`SELECT is_active, bind_password FROM ldap_sync_configs WHERE id = $1`, configID,
	).Scan(&isActive, &encryptedPwd)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "konfigurasi LDAP tidak ditemukan"})
		return
	}
	if !isActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "konfigurasi LDAP tidak aktif"})
		return
	}

	// Decrypt password (digunakan oleh LDAP agent saat bind — stub di sini)
	if encryptedPwd != nil && *encryptedPwd != "" {
		if _, err := security.DecryptLDAP(*encryptedPwd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mendekripsi bind_password: " + err.Error()})
			return
		}
		// plainPwd digunakan oleh LDAP dial (production agent)
	}

	// Cek apakah sudah ada sync yang running
	var runningCount int
	_ = database.Pool.QueryRow(c, `
		SELECT count(*) FROM ldap_sync_logs WHERE config_id = $1 AND status = 'running'
	`, configID).Scan(&runningCount)
	if runningCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "sync sedang berjalan, tunggu selesai"})
		return
	}

	var logID int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO ldap_sync_logs (config_id, status, triggered_by)
		VALUES ($1, 'running', $2) RETURNING id
	`, configID, actor).Scan(&logID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Simulasi sync selesai (stub — production: panggil LDAP agent/worker)
	go func() {
		_, _ = database.Pool.Exec(nil, `
			UPDATE ldap_sync_logs SET
				status       = 'success',
				users_found  = 0,
				users_synced = 0,
				errors       = '[]',
				finished_at  = now()
			WHERE id = $1
		`, logID)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"log_id":  logID,
		"message": "LDAP sync dijadwalkan (stub mode — agent LDAP diperlukan untuk sync aktual)",
	})
}

// GetLDAPSyncLogs mengembalikan log sinkronisasi LDAP.
func GetLDAPSyncLogs(c *gin.Context) {
	configID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	rows, err := database.Pool.Query(c, `
		SELECT l.id, l.config_id, l.status, l.users_found, l.users_synced, l.users_skipped,
		       l.errors, l.started_at, l.finished_at, l.triggered_by,
		       c.name AS config_name,
		       e.name AS triggered_by_name
		FROM ldap_sync_logs l
		JOIN ldap_sync_configs c ON c.id = l.config_id
		LEFT JOIN employees e    ON e.id = l.triggered_by
		WHERE l.config_id = $1
		ORDER BY l.started_at DESC
		LIMIT 100
	`, configID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.LDAPSyncLog
	for rows.Next() {
		var l models.LDAPSyncLog
		if err := rows.Scan(&l.ID, &l.ConfigID, &l.Status,
			&l.UsersFound, &l.UsersSynced, &l.UsersSkipped,
			&l.Errors, &l.StartedAt, &l.FinishedAt, &l.TriggeredBy,
			&l.ConfigName, &l.TriggeredByName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, l)
	}
	if list == nil {
		list = []models.LDAPSyncLog{}
	}
	c.JSON(http.StatusOK, list)
}

// DeleteLDAPConfig menghapus konfigurasi LDAP.
func DeleteLDAPConfig(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// Validasi tidak ada sync yang running
	var running int
	_ = database.Pool.QueryRow(c, `SELECT count(*) FROM ldap_sync_logs WHERE config_id=$1 AND status='running'`, id).Scan(&running)
	if running > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tidak bisa hapus config yang sedang sync"})
		return
	}
	_, err := database.Pool.Exec(c, `DELETE FROM ldap_sync_configs WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "konfigurasi LDAP berhasil dihapus"})
}

// keep time import used
var _ = time.Now
