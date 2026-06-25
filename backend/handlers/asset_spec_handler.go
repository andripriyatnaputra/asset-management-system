package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ASSET SPECIFICATIONS
// ============================================================

// GetAssetSpecification godoc
// @Summary Ambil spesifikasi hardware suatu aset
// @Tags Asset Specifications
// @Produce json
// @Param id path int true "Asset ID"
// @Success 200 {object} models.AssetSpecification
// @Router /assets/{id}/spec [get]
func GetAssetSpecification(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var spec models.AssetSpecification
	err := database.Pool.QueryRow(c, `
		SELECT
			s.id, s.asset_id,
			s.cpu_model, s.cpu_cores, s.cpu_speed_ghz,
			s.ram_gb, s.ram_type,
			s.storage_gb, s.storage_type,
			s.screen_size_inch, s.resolution,
			s.mac_address, s.ip_address,
			s.bios_version, s.firmware_version,
			s.os_name, s.os_version, s.os_license_key,
			s.form_factor, s.color, s.weight_kg,
			s.last_scanned_at, s.created_by, s.created_at, s.updated_at,
			a.name AS asset_name, a.asset_tag
		FROM asset_specifications s
		JOIN assets a ON a.id = s.asset_id
		WHERE s.asset_id = $1
	`, assetID).Scan(
		&spec.ID, &spec.AssetID,
		&spec.CPUModel, &spec.CPUCores, &spec.CPUSpeedGHz,
		&spec.RAMGB, &spec.RAMType,
		&spec.StorageGB, &spec.StorageType,
		&spec.ScreenSizeInch, &spec.Resolution,
		&spec.MACAddress, &spec.IPAddress,
		&spec.BIOSVersion, &spec.FirmwareVersion,
		&spec.OSName, &spec.OSVersion, &spec.OSLicenseKey,
		&spec.FormFactor, &spec.Color, &spec.WeightKG,
		&spec.LastScannedAt, &spec.CreatedBy, &spec.CreatedAt, &spec.UpdatedAt,
		&spec.AssetName, &spec.AssetTag,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "spesifikasi tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, spec)
}

// UpsertAssetSpecification godoc
// @Summary Buat atau update spesifikasi hardware aset (upsert)
// @Tags Asset Specifications
// @Accept json
// @Produce json
// @Param id path int true "Asset ID"
// @Router /assets/{id}/spec [put]
func UpsertAssetSpecification(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		CPUModel        *string  `json:"cpu_model"`
		CPUCores        *int     `json:"cpu_cores"`
		CPUSpeedGHz     *float64 `json:"cpu_speed_ghz"`
		RAMGB           *float64 `json:"ram_gb"`
		RAMType         *string  `json:"ram_type"`
		StorageGB       *float64 `json:"storage_gb"`
		StorageType     *string  `json:"storage_type"`
		ScreenSizeInch  *float64 `json:"screen_size_inch"`
		Resolution      *string  `json:"resolution"`
		MACAddress      *string  `json:"mac_address"`
		IPAddress       *string  `json:"ip_address"`
		BIOSVersion     *string  `json:"bios_version"`
		FirmwareVersion *string  `json:"firmware_version"`
		OSName          *string  `json:"os_name"`
		OSVersion       *string  `json:"os_version"`
		OSLicenseKey    *string  `json:"os_license_key"`
		FormFactor      *string  `json:"form_factor"`
		Color           *string  `json:"color"`
		WeightKG        *float64 `json:"weight_kg"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	_, err := database.Pool.Exec(c, `
		INSERT INTO asset_specifications (
			asset_id, cpu_model, cpu_cores, cpu_speed_ghz,
			ram_gb, ram_type, storage_gb, storage_type,
			screen_size_inch, resolution, mac_address, ip_address,
			bios_version, firmware_version, os_name, os_version, os_license_key,
			form_factor, color, weight_kg, last_scanned_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
		ON CONFLICT (asset_id) DO UPDATE SET
			cpu_model        = COALESCE(EXCLUDED.cpu_model,        asset_specifications.cpu_model),
			cpu_cores        = COALESCE(EXCLUDED.cpu_cores,        asset_specifications.cpu_cores),
			cpu_speed_ghz    = COALESCE(EXCLUDED.cpu_speed_ghz,    asset_specifications.cpu_speed_ghz),
			ram_gb           = COALESCE(EXCLUDED.ram_gb,           asset_specifications.ram_gb),
			ram_type         = COALESCE(EXCLUDED.ram_type,         asset_specifications.ram_type),
			storage_gb       = COALESCE(EXCLUDED.storage_gb,       asset_specifications.storage_gb),
			storage_type     = COALESCE(EXCLUDED.storage_type,     asset_specifications.storage_type),
			screen_size_inch = COALESCE(EXCLUDED.screen_size_inch, asset_specifications.screen_size_inch),
			resolution       = COALESCE(EXCLUDED.resolution,       asset_specifications.resolution),
			mac_address      = COALESCE(EXCLUDED.mac_address,      asset_specifications.mac_address),
			ip_address       = COALESCE(EXCLUDED.ip_address,       asset_specifications.ip_address),
			bios_version     = COALESCE(EXCLUDED.bios_version,     asset_specifications.bios_version),
			firmware_version = COALESCE(EXCLUDED.firmware_version, asset_specifications.firmware_version),
			os_name          = COALESCE(EXCLUDED.os_name,          asset_specifications.os_name),
			os_version       = COALESCE(EXCLUDED.os_version,       asset_specifications.os_version),
			os_license_key   = COALESCE(EXCLUDED.os_license_key,   asset_specifications.os_license_key),
			form_factor      = COALESCE(EXCLUDED.form_factor,      asset_specifications.form_factor),
			color            = COALESCE(EXCLUDED.color,            asset_specifications.color),
			weight_kg        = COALESCE(EXCLUDED.weight_kg,        asset_specifications.weight_kg),
			last_scanned_at  = EXCLUDED.last_scanned_at,
			updated_at       = now()
	`, assetID,
		req.CPUModel, req.CPUCores, req.CPUSpeedGHz,
		req.RAMGB, req.RAMType, req.StorageGB, req.StorageType,
		req.ScreenSizeInch, req.Resolution, req.MACAddress, req.IPAddress,
		req.BIOSVersion, req.FirmwareVersion, req.OSName, req.OSVersion, req.OSLicenseKey,
		req.FormFactor, req.Color, req.WeightKG, now, actor,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "spesifikasi aset berhasil disimpan"})
}

// ============================================================
// SOFTWARE USAGE LOGS
// ============================================================

// LogSoftwareUsage godoc
// @Summary Catat sesi penggunaan lisensi (manual / import)
// @Tags Software Usage
// @Accept json
// @Produce json
// @Router /licenses/{id}/usage [post]
func LogSoftwareUsage(c *gin.Context) {
	licenseID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		AssetID      *int64     `json:"asset_id"`
		EmployeeID   *int64     `json:"employee_id"`
		SessionStart time.Time  `json:"session_start" binding:"required"`
		SessionEnd   *time.Time `json:"session_end"`
		UsageMinutes *int       `json:"usage_minutes"`
		Source       string     `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Source == "" {
		req.Source = "manual"
	}

	// Auto-calculate usage_minutes dari session_start & session_end
	if req.UsageMinutes == nil && req.SessionEnd != nil {
		mins := int(req.SessionEnd.Sub(req.SessionStart).Minutes())
		req.UsageMinutes = &mins
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO software_usage_logs
			(license_id, asset_id, employee_id, session_start, session_end, usage_minutes, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id
	`, licenseID, req.AssetID, req.EmployeeID,
		req.SessionStart, req.SessionEnd, req.UsageMinutes, req.Source,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "usage log berhasil dicatat"})
}

// GetSoftwareUsageLogs godoc
// @Summary List usage logs untuk suatu lisensi
// @Tags Software Usage
// @Produce json
// @Param id path int true "License ID"
// @Router /licenses/{id}/usage [get]
func GetSoftwareUsageLogs(c *gin.Context) {
	licenseID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	rows, err := database.Pool.Query(c, `
		SELECT
			sul.id, sul.license_id, sul.asset_id, sul.employee_id,
			sul.session_start, sul.session_end, sul.usage_minutes, sul.source, sul.created_at,
			l.name  AS license_name,
			a.name  AS asset_name,
			e.name  AS employee_name
		FROM software_usage_logs sul
		JOIN licenses l        ON l.id = sul.license_id
		LEFT JOIN assets    a  ON a.id = sul.asset_id
		LEFT JOIN employees e  ON e.id = sul.employee_id
		WHERE sul.license_id = $1
		ORDER BY sul.session_start DESC
		LIMIT 500
	`, licenseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.SoftwareUsageLog
	for rows.Next() {
		var u models.SoftwareUsageLog
		if err := rows.Scan(
			&u.ID, &u.LicenseID, &u.AssetID, &u.EmployeeID,
			&u.SessionStart, &u.SessionEnd, &u.UsageMinutes, &u.Source, &u.CreatedAt,
			&u.LicenseName, &u.AssetName, &u.EmployeeName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, u)
	}
	if list == nil {
		list = []models.SoftwareUsageLog{}
	}
	c.JSON(http.StatusOK, list)
}

// GetLicenseReconciliation godoc
// @Summary Laporan rekonsiliasi lisensi (entitlement vs aktual)
// @Tags Software Usage
// @Produce json
// @Param status query string false "Filter: compliant|over_licensed|under_utilized"
// @Router /licenses/reconciliation [get]
func GetLicenseReconciliation(c *gin.Context) {
	statusFilter := c.Query("status")

	query := `
		SELECT license_id, license_name, license_type, license_model,
		       entitled_seats, installed_seats, available_seats,
		       reconciliation_status, expiration_date, compliance_status,
		       vendor, cost, currency, active_users_90d, last_used_at
		FROM v_license_reconciliation
	`
	args := []interface{}{}
	if statusFilter != "" {
		query += " WHERE reconciliation_status = $1"
		args = append(args, statusFilter)
	}
	query += " ORDER BY reconciliation_status, license_name"

	rows, err := database.Pool.Query(c, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.LicenseReconciliation
	for rows.Next() {
		var r models.LicenseReconciliation
		if err := rows.Scan(
			&r.LicenseID, &r.LicenseName, &r.LicenseType, &r.LicenseModel,
			&r.EntitledSeats, &r.InstalledSeats, &r.AvailableSeats,
			&r.ReconciliationStatus, &r.ExpirationDate, &r.ComplianceStatus,
			&r.Vendor, &r.Cost, &r.Currency, &r.ActiveUsers90d, &r.LastUsedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, r)
	}
	if list == nil {
		list = []models.LicenseReconciliation{}
	}
	c.JSON(http.StatusOK, list)
}

// ============================================================
// ASSET DISPOSAL RECORDS
// ============================================================

// GetDisposalRecord godoc
// @Summary Ambil record disposal suatu aset
// @Tags Asset Disposal
// @Produce json
// @Param id path int true "Asset ID"
// @Router /assets/{id}/disposal [get]
func GetDisposalRecord(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var d models.AssetDisposalRecord
	err := database.Pool.QueryRow(c, `
		SELECT
			d.id, d.asset_id, d.disposal_method, d.data_wipe_method,
			d.data_wipe_completed, d.certificate_number, d.certificate_url,
			d.environmental_compliant, d.regulatory_notes, d.vendor,
			d.disposal_value, d.authorization_by, d.executed_by,
			d.date_disposed, d.created_by, d.created_at, d.updated_at,
			a.name AS asset_name, a.asset_tag,
			auth.name AS authorized_by_name,
			exec.name AS executed_by_name
		FROM asset_disposal_records d
		JOIN assets a         ON a.id    = d.asset_id
		JOIN employees auth   ON auth.id = d.authorization_by
		LEFT JOIN employees exec ON exec.id = d.executed_by
		WHERE d.asset_id = $1
	`, assetID).Scan(
		&d.ID, &d.AssetID, &d.DisposalMethod, &d.DataWipeMethod,
		&d.DataWipeCompleted, &d.CertificateNumber, &d.CertificateURL,
		&d.EnvironmentalCompliant, &d.RegulatoryNotes, &d.Vendor,
		&d.DisposalValue, &d.AuthorizationBy, &d.ExecutedBy,
		&d.DateDisposed, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
		&d.AssetName, &d.AssetTag,
		&d.AuthorizedByName, &d.ExecutedByName,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "disposal record tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, d)
}

// GetAllDisposalRecords godoc
// @Summary List semua disposal records
// @Tags Asset Disposal
// @Produce json
// @Param env_compliant query bool false "Filter yang sudah environmental compliant"
// @Router /assets/disposals [get]
func GetAllDisposalRecords(c *gin.Context) {
	envFilter := c.Query("env_compliant")

	query := `
		SELECT
			d.id, d.asset_id, d.disposal_method, d.data_wipe_method,
			d.data_wipe_completed, d.certificate_number, d.certificate_url,
			d.environmental_compliant, d.regulatory_notes, d.vendor,
			d.disposal_value, d.authorization_by, d.executed_by,
			d.date_disposed, d.created_by, d.created_at, d.updated_at,
			a.name, a.asset_tag,
			auth.name, exec.name
		FROM asset_disposal_records d
		JOIN assets a            ON a.id    = d.asset_id
		JOIN employees auth      ON auth.id = d.authorization_by
		LEFT JOIN employees exec ON exec.id = d.executed_by
		WHERE 1=1
	`
	args := []interface{}{}
	idx := 1
	if envFilter == "true" {
		query += " AND d.environmental_compliant = true"
	} else if envFilter == "false" {
		query += " AND d.environmental_compliant = false"
	}
	_ = idx

	query += " ORDER BY d.date_disposed DESC"

	rows, err := database.Pool.Query(c, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.AssetDisposalRecord
	for rows.Next() {
		var d models.AssetDisposalRecord
		if err := rows.Scan(
			&d.ID, &d.AssetID, &d.DisposalMethod, &d.DataWipeMethod,
			&d.DataWipeCompleted, &d.CertificateNumber, &d.CertificateURL,
			&d.EnvironmentalCompliant, &d.RegulatoryNotes, &d.Vendor,
			&d.DisposalValue, &d.AuthorizationBy, &d.ExecutedBy,
			&d.DateDisposed, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
			&d.AssetName, &d.AssetTag,
			&d.AuthorizedByName, &d.ExecutedByName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, d)
	}
	if list == nil {
		list = []models.AssetDisposalRecord{}
	}
	c.JSON(http.StatusOK, list)
}

// CreateDisposalRecord godoc
// @Summary Buat disposal record untuk aset yang akan/sudah dibuang
// @Tags Asset Disposal
// @Accept json
// @Produce json
// @Param id path int true "Asset ID"
// @Router /assets/{id}/disposal [post]
func CreateDisposalRecord(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		DisposalMethod         string   `json:"disposal_method" binding:"required"`
		DataWipeMethod         *string  `json:"data_wipe_method"`
		DataWipeCompleted      bool     `json:"data_wipe_completed"`
		CertificateNumber      *string  `json:"certificate_number"`
		CertificateURL         *string  `json:"certificate_url"`
		EnvironmentalCompliant bool     `json:"environmental_compliant"`
		RegulatoryNotes        *string  `json:"regulatory_notes"`
		Vendor                 *string  `json:"vendor"`
		DisposalValue          *float64 `json:"disposal_value"`
		AuthorizationBy        int64    `json:"authorization_by" binding:"required"`
		ExecutedBy             *int64   `json:"executed_by"`
		DateDisposed           string   `json:"date_disposed" binding:"required"` // YYYY-MM-DD
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dateDisposed, err := time.Parse("2006-01-02", req.DateDisposed)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format date_disposed harus YYYY-MM-DD"})
		return
	}

	var id int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO asset_disposal_records (
			asset_id, disposal_method, data_wipe_method, data_wipe_completed,
			certificate_number, certificate_url, environmental_compliant,
			regulatory_notes, vendor, disposal_value,
			authorization_by, executed_by, date_disposed, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id
	`, assetID, req.DisposalMethod, req.DataWipeMethod, req.DataWipeCompleted,
		req.CertificateNumber, req.CertificateURL, req.EnvironmentalCompliant,
		req.RegulatoryNotes, req.Vendor, req.DisposalValue,
		req.AuthorizationBy, req.ExecutedBy, dateDisposed, actor,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update asset lifecycle_stage → disposal_approved
	_, _ = database.Pool.Exec(c, `
		UPDATE assets SET lifecycle_stage = 'disposal_approved', updated_at = now()
		WHERE id = $1
	`, assetID)

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "disposal record berhasil dibuat"})
}

// UpdateDisposalRecord godoc
// @Summary Update disposal record (sertifikat, environmental compliance, dll)
// @Tags Asset Disposal
// @Param id path int true "Asset ID"
// @Router /assets/{id}/disposal [put]
func UpdateDisposalRecord(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		DataWipeCompleted      *bool    `json:"data_wipe_completed"`
		CertificateNumber      *string  `json:"certificate_number"`
		CertificateURL         *string  `json:"certificate_url"`
		EnvironmentalCompliant *bool    `json:"environmental_compliant"`
		RegulatoryNotes        *string  `json:"regulatory_notes"`
		DisposalValue          *float64 `json:"disposal_value"`
		ExecutedBy             *int64   `json:"executed_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE asset_disposal_records SET
			data_wipe_completed     = COALESCE($1, data_wipe_completed),
			certificate_number      = COALESCE($2, certificate_number),
			certificate_url         = COALESCE($3, certificate_url),
			environmental_compliant = COALESCE($4, environmental_compliant),
			regulatory_notes        = COALESCE($5, regulatory_notes),
			disposal_value          = COALESCE($6, disposal_value),
			executed_by             = COALESCE($7, executed_by),
			updated_at              = now()
		WHERE asset_id = $8
	`, req.DataWipeCompleted, req.CertificateNumber, req.CertificateURL,
		req.EnvironmentalCompliant, req.RegulatoryNotes, req.DisposalValue,
		req.ExecutedBy, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "disposal record berhasil diupdate"})
}
