// File: backend/handlers/license_handler.go
package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func parseDatePtr(s *string) *time.Time {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	layouts := []string{"2006-01-02", time.RFC3339}
	for _, l := range layouts {
		if t, err := time.Parse(l, *s); err == nil {
			return &t
		}
	}
	return nil
}

// =============================================================
// 🆕 CREATE LICENSE (POST /licenses)
// =============================================================
func CreateLicense(c *gin.Context) {
	var req struct {
		Name              string   `json:"name" binding:"required"`
		LicenseKey        *string  `json:"license_key"`
		Vendor            *string  `json:"vendor"`
		Publisher         *string  `json:"publisher"`
		Version           *string  `json:"version"`
		LicenseType       *string  `json:"license_type"`
		LicenseModel      *string  `json:"license_model"`
		Metric            *string  `json:"metric"`
		Category          *string  `json:"category"`
		BudgetID          *int64   `json:"budget_id"`
		ContractID        *int64   `json:"contract_id"`
		TotalSeats        int      `json:"total_seats"`
		Cost              *float64 `json:"cost"`
		PurchaseDate      *string  `json:"purchase_date"`
		ExpirationDate    *string  `json:"expiration_date"`
		MaintenanceExpiry *string  `json:"maintenance_expiry"`
		Currency          *string  `json:"currency"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "detail": err.Error()})
		return
	}

	// 🔹 License key must be unique
	if req.LicenseKey != nil {
		var exists bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM licenses WHERE license_key=$1 AND deleted_at IS NULL)`,
			req.LicenseKey).Scan(&exists)
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "license_key already exists"})
			return
		}
	}

	// 🔹 Validate contract_id
	if req.ContractID != nil {
		var exists bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM contracts WHERE id=$1 AND deleted_at IS NULL)`,
			*req.ContractID,
		).Scan(&exists)
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contract_id not found"})
			return
		}
	}

	// 🔹 Validate budget_id
	if req.BudgetID != nil {
		var exists bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM budgets WHERE id=$1 AND deleted_at IS NULL)`,
			*req.BudgetID,
		).Scan(&exists)
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "budget_id not found"})
			return
		}
	}

	// 🔹 Strong governance default currency
	curr := "IDR"
	if req.Currency != nil && *req.Currency != "" {
		curr = *req.Currency
	}

	now := time.Now()
	purchase := parseDatePtr(req.PurchaseDate)
	expire := parseDatePtr(req.ExpirationDate)
	maint := parseDatePtr(req.MaintenanceExpiry)
	if purchase == nil {
		purchase = &now
	}

	var createdBy *int64
	if uid, ok := c.Get("userID"); ok {
		tmp := uid.(int64)
		createdBy = &tmp
	}

	// =========================================================
	// INSERT
	// =========================================================
	var id int64
	err := database.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO licenses (
			name, license_key, vendor, publisher, version,
			license_type, license_model, metric, category,
			budget_id, contract_id, total_seats, cost,
			purchase_date, expiration_date, maintenance_expiry,
			currency, created_by, updated_by,
			created_at, updated_at
		)
		VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,
			$10,$11,$12,$13,
			$14,$15,$16,
			$17,$18,$19,
			NOW(),NOW()
		)
		RETURNING id;
	`,
		req.Name, req.LicenseKey, req.Vendor, req.Publisher, req.Version,
		req.LicenseType, req.LicenseModel, req.Metric, req.Category,
		req.BudgetID, req.ContractID, req.TotalSeats, req.Cost,
		purchase, expire, maint,
		curr, createdBy, createdBy,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create license", "detail": err.Error()})
		return
	}

	// 🔹 Governance score baseline
	score := governanceScore(req.ContractID != nil, req.BudgetID != nil, true)
	database.Pool.Exec(c.Request.Context(),
		`UPDATE licenses SET compliance_score=$1 WHERE id=$2`, score, id)

	middleware.LogAction(c, "licenses", id, "CREATE", req)

	c.JSON(http.StatusCreated, gin.H{
		"id":      id,
		"message": "License created successfully (A++)",
	})
}

// =============================================================
// 📋 GET ALL LICENSES (sorted, deterministic, safe)
// =============================================================
func GetAllLicenses(c *gin.Context) {
	search := strings.TrimSpace(strings.ToLower(c.Query("q")))

	base := `
		SELECT 
			l.id,
			l.name,
			l.license_key,
			l.vendor,
			l.license_type,
			l.license_model,
			l.total_seats,
			COALESCE(used.used_count, 0) AS used_seats,
			l.compliance_status,
			l.compliance_score,
			l.purchase_date,
			l.expiration_date,
			l.cost,
			l.contract_id,
			c.contract_number,
			l.created_at
		FROM licenses l
		LEFT JOIN (
			SELECT license_id, COUNT(*) AS used_count
			  FROM software_installations
			 WHERE removed_at IS NULL
			 GROUP BY license_id
		) used ON used.license_id = l.id
		LEFT JOIN contracts c ON c.id = l.contract_id
		WHERE l.deleted_at IS NULL
	`

	var rows pgx.Rows
	var err error

	if search != "" {
		rows, err = database.Pool.Query(c.Request.Context(),
			base+` AND (LOWER(l.name) LIKE '%'||$1||'%' OR LOWER(l.vendor) LIKE '%'||$1||'%')
			       ORDER BY l.name ASC`,
			search)
	} else {
		rows, err = database.Pool.Query(c.Request.Context(),
			base+` ORDER BY l.name ASC`)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch licenses", "detail": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.License
	for rows.Next() {
		var l models.License
		if err := rows.Scan(
			&l.ID,
			&l.Name,
			&l.LicenseKey,
			&l.Vendor,
			&l.LicenseType,
			&l.LicenseModel,
			&l.TotalSeats,
			&l.UsedSeats,
			&l.ComplianceStatus,
			&l.ComplianceScore,
			&l.PurchaseDate,
			&l.ExpirationDate,
			&l.Cost,
			&l.ContractID,
			&l.ContractNumber,
			&l.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan license row", "detail": err.Error()})
			return
		}
		list = append(list, l)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// =============================================================
// 🔍 GET LICENSE BY ID
// =============================================================
func GetLicenseByID(c *gin.Context) {
	id := c.Param("id")
	var l models.License

	query := `
		SELECT id, name, license_key, vendor, publisher, version,
		       license_type, license_model, metric, category,
		       total_seats, cost, purchase_date, expiration_date,
		       maintenance_expiry, compliance_score,
		       contract_id, budget_id, created_at, updated_at
		  FROM licenses
		 WHERE id=$1 AND deleted_at IS NULL
	`
	err := database.Pool.QueryRow(c.Request.Context(), query, id).Scan(
		&l.ID, &l.Name, &l.LicenseKey, &l.Vendor, &l.Publisher, &l.Version,
		&l.LicenseType, &l.LicenseModel, &l.Metric, &l.Category,
		&l.TotalSeats, &l.Cost, &l.PurchaseDate, &l.ExpirationDate,
		&l.MaintenanceExpiry, &l.ComplianceScore,
		&l.ContractID, &l.BudgetID, &l.CreatedAt, &l.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// ==========================================================
	// Health score by age
	// ==========================================================
	var health float64 = 100
	if l.ExpirationDate != nil {
		total := l.ExpirationDate.Sub(l.CreatedAt).Hours() / 24
		left := time.Until(*l.ExpirationDate).Hours() / 24
		if total > 0 {
			health = (left / total) * 100
			if health < 0 {
				health = 0
			}
		}
	}

	// deterministic governance score
	gov := governanceScore(
		l.ContractID != nil,
		l.BudgetID != nil,
		l.ExpirationDate != nil,
	)

	c.JSON(http.StatusOK, gin.H{
		"license":        l,
		"license_health": fmt.Sprintf("%.1f%%", health),
		"governance":     gov,
	})
}

// =============================================================
// ✏️ UPDATE LICENSE (PATCH /licenses/:id)
// =============================================================
func UpdateLicense(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		Name              *string  `json:"name,omitempty"`
		LicenseKey        *string  `json:"license_key,omitempty"`
		Vendor            *string  `json:"vendor,omitempty"`
		Publisher         *string  `json:"publisher,omitempty"`
		Version           *string  `json:"version,omitempty"`
		LicenseType       *string  `json:"license_type,omitempty"`
		LicenseModel      *string  `json:"license_model,omitempty"`
		Metric            *string  `json:"metric,omitempty"`
		Category          *string  `json:"category,omitempty"`
		BudgetID          **int64  `json:"budget_id,omitempty"`
		ContractID        **int64  `json:"contract_id,omitempty"`
		TotalSeats        *int     `json:"total_seats,omitempty"`
		Cost              *float64 `json:"cost,omitempty"`
		PurchaseDate      *string  `json:"purchase_date,omitempty"`
		ExpirationDate    *string  `json:"expiration_date,omitempty"`
		MaintenanceExpiry *string  `json:"maintenance_expiry,omitempty"`
		Currency          *string  `json:"currency,omitempty"`
		EntitlementDoc    *string  `json:"entitlement_doc,omitempty"`
		ProcRef           *string  `json:"procurement_reference,omitempty"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "detail": err.Error()})
		return
	}

	// 🔹 Validasi license_key unik (kalau diubah)
	if body.LicenseKey != nil {
		var exists bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM licenses 
			                WHERE license_key=$1 AND id<>$2 AND deleted_at IS NULL)`,
			*body.LicenseKey, id).Scan(&exists)
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "license_key already in use"})
			return
		}
	}

	// 🔹 Validasi budget_id / contract_id jika diubah (body.BudgetID / ContractID double pointer)
	if body.BudgetID != nil {
		if *body.BudgetID != nil {
			var exists bool
			_ = database.Pool.QueryRow(c.Request.Context(),
				`SELECT EXISTS(SELECT 1 FROM budgets WHERE id=$1 AND deleted_at IS NULL)`,
				**body.BudgetID).Scan(&exists)
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "budget_id not found"})
				return
			}
		}
	}
	if body.ContractID != nil {
		if *body.ContractID != nil {
			var exists bool
			_ = database.Pool.QueryRow(c.Request.Context(),
				`SELECT EXISTS(SELECT 1 FROM contracts WHERE id=$1 AND deleted_at IS NULL)`,
				**body.ContractID).Scan(&exists)
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "contract_id not found"})
				return
			}
		}
	}

	// 🔹 Build dynamic query
	setParts := []string{}
	args := []any{}
	i := 1

	if body.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name=$%d", i))
		args = append(args, *body.Name)
		i++
	}
	if body.LicenseKey != nil {
		setParts = append(setParts, fmt.Sprintf("license_key=$%d", i))
		args = append(args, *body.LicenseKey)
		i++
	}
	if body.Vendor != nil {
		setParts = append(setParts, fmt.Sprintf("vendor=$%d", i))
		args = append(args, *body.Vendor)
		i++
	}
	if body.Publisher != nil {
		setParts = append(setParts, fmt.Sprintf("publisher=$%d", i))
		args = append(args, *body.Publisher)
		i++
	}
	if body.Version != nil {
		setParts = append(setParts, fmt.Sprintf("version=$%d", i))
		args = append(args, *body.Version)
		i++
	}
	if body.LicenseType != nil {
		setParts = append(setParts, fmt.Sprintf("license_type=$%d", i))
		args = append(args, *body.LicenseType)
		i++
	}
	if body.LicenseModel != nil {
		setParts = append(setParts, fmt.Sprintf("license_model=$%d", i))
		args = append(args, *body.LicenseModel)
		i++
	}
	if body.Metric != nil {
		setParts = append(setParts, fmt.Sprintf("metric=$%d", i))
		args = append(args, *body.Metric)
		i++
	}
	if body.Category != nil {
		setParts = append(setParts, fmt.Sprintf("category=$%d", i))
		args = append(args, *body.Category)
		i++
	}
	if body.TotalSeats != nil {
		setParts = append(setParts, fmt.Sprintf("total_seats=$%d", i))
		args = append(args, *body.TotalSeats)
		i++
	}
	if body.Cost != nil {
		setParts = append(setParts, fmt.Sprintf("cost=$%d", i))
		args = append(args, *body.Cost)
		i++
	}
	if body.PurchaseDate != nil {
		setParts = append(setParts, fmt.Sprintf("purchase_date=$%d", i))
		args = append(args, parseDatePtr(body.PurchaseDate))
		i++
	}
	if body.ExpirationDate != nil {
		setParts = append(setParts, fmt.Sprintf("expiration_date=$%d", i))
		args = append(args, parseDatePtr(body.ExpirationDate))
		i++
	}
	if body.MaintenanceExpiry != nil {
		setParts = append(setParts, fmt.Sprintf("maintenance_expiry=$%d", i))
		args = append(args, parseDatePtr(body.MaintenanceExpiry))
		i++
	}
	if body.Currency != nil {
		setParts = append(setParts, fmt.Sprintf("currency=$%d", i))
		args = append(args, *body.Currency)
		i++
	}
	if body.EntitlementDoc != nil {
		setParts = append(setParts, fmt.Sprintf("entitlement_doc=$%d", i))
		args = append(args, *body.EntitlementDoc)
		i++
	}
	if body.ProcRef != nil {
		setParts = append(setParts, fmt.Sprintf("procurement_reference=$%d", i))
		args = append(args, *body.ProcRef)
		i++
	}
	if body.BudgetID != nil {
		setParts = append(setParts, fmt.Sprintf("budget_id=$%d", i))
		args = append(args, *body.BudgetID)
		i++
	}
	if body.ContractID != nil {
		setParts = append(setParts, fmt.Sprintf("contract_id=$%d", i))
		args = append(args, *body.ContractID)
		i++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	// updated_at
	setParts = append(setParts, "updated_at=NOW()")

	query := fmt.Sprintf(`UPDATE licenses SET %s WHERE id=$%d AND deleted_at IS NULL`,
		strings.Join(setParts, ","), i)
	args = append(args, id)

	cmd, err := database.Pool.Exec(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update license", "detail": err.Error()})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	middleware.LogAction(c, "licenses", mustAtoi64(id), "UPDATE", body)
	c.JSON(http.StatusOK, gin.H{"message": "License updated successfully"})
}

// =============================================================
// 🗑 DELETE LICENSE (soft delete + OPEX reversal)
// =============================================================
func DeleteLicense(c *gin.Context) {
	idStr := c.Param("id")
	id := mustAtoi64(idStr)

	// 🔹 Block if still installed on assets
	var installCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) 
		   FROM software_installations 
		  WHERE license_id=$1 AND removed_at IS NULL`, id).
		Scan(&installCount)

	if installCount > 0 {
		middleware.LogAction(c, "licenses", id, "DELETE_BLOCKED",
			gin.H{"active_installations": installCount})
		c.JSON(http.StatusForbidden, gin.H{
			"error":                "cannot delete license; software still installed",
			"active_installations": installCount,
		})
		return
	}

	// 🔹 Get financial info
	var (
		budgetID *int64
		cost     *float64
		name     *string
		curr     *string
	)
	err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT budget_id, cost, name, currency
		   FROM licenses
		  WHERE id=$1 AND deleted_at IS NULL`, id).
		Scan(&budgetID, &cost, &name, &curr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// 🔹 Soft delete
	_, err = database.Pool.Exec(c.Request.Context(),
		`UPDATE licenses SET deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}

	middleware.LogAction(c, "licenses", id, "DELETE", nil)

	// 🔹 OPEX reversal (if has budget & cost)
	if budgetID != nil && cost != nil && *cost > 0 {
		note := fmt.Sprintf("Reversal OPEX license %s", ptrOr(name, fmt.Sprintf("ID %d", id)))
		currency := ptrOr(curr, "IDR")

		_ = CreateBudgetTransaction(c.Request.Context(), BudgetTxInput{
			BudgetID:   *budgetID,
			EntityType: "license",
			EntityID:   &id,
			Amount:     -1 * *cost,
			Category:   strPtr("OPEX"),
			Currency:   &currency,
			// CostCenterID intentionally nil – license.cost_center is text, not FK
			CreatedBy: getUserIDPtr(c),
			Notes:     note,
		})
		_ = RecalculateBudgetTotals(c.Request.Context(), *budgetID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "license deleted"})
}

// =============================================================
// 📦 GET SOFTWARE INSTALLED ON ASSET
// =============================================================
func GetSoftwareForAsset(c *gin.Context) {
	assetID := c.Param("id")
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT si.id, l.name, l.license_key, si.installation_date
		  FROM software_installations si
		  JOIN licenses l ON l.id = si.license_id
		 WHERE si.asset_id=$1 AND si.removed_at IS NULL
		 ORDER BY si.installation_date DESC`, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch installed software"})
		return
	}
	defer rows.Close()

	var list []map[string]any
	for rows.Next() {
		var id int64
		var name, key *string
		var date time.Time
		rows.Scan(&id, &name, &key, &date)
		list = append(list, gin.H{
			"installation_id":   id,
			"license_name":      name,
			"license_key":       key,
			"installation_date": date,
		})
	}
	c.JSON(http.StatusOK, list)
}

// =============================================================
// 📥 INSTALL SOFTWARE TO ASSET (seat-aware)
// =============================================================
func InstallSoftwareOnAsset(c *gin.Context) {
	assetID := c.Param("id")
	var body struct {
		LicenseID int64 `json:"license_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	ctx := c.Request.Context()

	// 🔹 Pastikan asset ada & tidak dihapus
	var assetExists bool
	_ = database.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND deleted_at IS NULL)`,
		assetID).Scan(&assetExists)
	if !assetExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}

	// 🔹 Pastikan kombinasi belum terpasang
	var count int
	_ = database.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM software_installations 
		  WHERE asset_id=$1 AND license_id=$2 AND removed_at IS NULL`,
		assetID, body.LicenseID).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Software already installed on this asset"})
		return
	}

	// 🔹 Pastikan tidak melebihi total_seats
	var total, used int
	err := database.Pool.QueryRow(ctx, `
		SELECT total_seats,
		       COALESCE((
				SELECT COUNT(*) 
				FROM software_installations 
				WHERE license_id=$1 AND removed_at IS NULL
			),0) AS used
		FROM licenses 
		WHERE id=$1 AND deleted_at IS NULL`,
		body.LicenseID).Scan(&total, &used)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
		return
	}

	if total > 0 && used >= total {
		c.JSON(http.StatusConflict, gin.H{"error": "License seats exceeded"})
		return
	}

	// 🔹 Insert installation
	_, err = database.Pool.Exec(ctx,
		`INSERT INTO software_installations (asset_id, license_id, installation_date, removed_at)
		 VALUES ($1,$2,NOW(),NULL)`,
		assetID, body.LicenseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to install software"})
		return
	}

	middleware.LogAction(c, "software_installations", 0, "INSTALL",
		gin.H{"asset_id": assetID, "license_id": body.LicenseID})

	c.JSON(http.StatusOK, gin.H{"message": "Software installed successfully"})
}

// =============================================================
// 📤 UNINSTALL SOFTWARE (soft remove via removed_at)
// =============================================================
func UninstallSoftwareFromAsset(c *gin.Context) {
	assetID := c.Param("id")
	installID := c.Param("installation_id")

	var licenseID int64
	err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT license_id FROM software_installations 
		  WHERE id=$1 AND asset_id=$2 AND removed_at IS NULL`,
		installID, assetID).Scan(&licenseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Installation not found"})
		return
	}

	_, err = database.Pool.Exec(c.Request.Context(),
		`UPDATE software_installations 
		    SET removed_at=NOW()
		  WHERE id=$1 AND asset_id=$2`, installID, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to uninstall software"})
		return
	}

	middleware.LogAction(c, "software_installations", mustAtoi64(installID), "UNINSTALL",
		gin.H{"asset_id": assetID, "license_id": licenseID})
	c.JSON(http.StatusOK, gin.H{"message": "Software uninstalled successfully"})
}

// =============================================================
// 📊 LICENSE COMPLIANCE (seat-based)
// =============================================================
func GetLicenseCompliance(c *gin.Context) {
	ctx := c.Request.Context()
	query := `
		SELECT l.id, l.name, l.vendor, l.license_type, l.license_model,
			   COALESCE(l.total_seats,0) AS total_seats,
			   COALESCE(used.used_count,0) AS used_seats
		  FROM licenses l
		  LEFT JOIN (
			  SELECT license_id, COUNT(*) AS used_count
			    FROM software_installations
			   WHERE removed_at IS NULL
			   GROUP BY license_id
		  ) used ON used.license_id = l.id
		 WHERE l.deleted_at IS NULL
		 ORDER BY l.name;
	`
	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch license compliance"})
		return
	}
	defer rows.Close()

	type record struct {
		ID           int64   `json:"id"`
		Name         string  `json:"name"`
		Vendor       *string `json:"vendor"`
		LicenseType  *string `json:"license_type"`
		LicenseModel *string `json:"license_model"`
		TotalSeats   int     `json:"total_seats"`
		UsedSeats    int     `json:"used_seats"`
		Status       string  `json:"status"`
		Score        float64 `json:"score"`
	}
	var list []record

	for rows.Next() {
		var r record
		rows.Scan(&r.ID, &r.Name, &r.Vendor, &r.LicenseType,
			&r.LicenseModel, &r.TotalSeats, &r.UsedSeats)

		if r.TotalSeats > 0 {
			// Score = persentase seats yang masih free
			free := r.TotalSeats - r.UsedSeats
			r.Score = math.Max(0, math.Min(100, float64(free)/float64(r.TotalSeats)*100))
		} else {
			r.Score = 0
		}

		if r.TotalSeats == 0 {
			r.Status = "unknown"
		} else if r.UsedSeats <= r.TotalSeats {
			r.Status = "compliant"
		} else {
			r.Status = "non-compliant"
		}
		list = append(list, r)
	}

	// 🔁 Update compliance_status di database
	for _, r := range list {
		_, _ = database.Pool.Exec(ctx,
			`UPDATE licenses 
			    SET compliance_status=$1,
			        compliance_score=$2,
			        verification_date=NOW(),
			        updated_at=NOW()
			  WHERE id=$3`,
			r.Status, r.Score, r.ID)
	}

	// 🔔 Broadcast alert non-compliant
	for _, r := range list {
		if r.Status == "non-compliant" {
			msg := fmt.Sprintf("License %s overused (%d/%d seats)", r.Name, r.UsedSeats, r.TotalSeats)
			services.BroadcastAlert(msg, "warning")
		}
	}

	c.JSON(http.StatusOK, gin.H{"compliance": list})
}
