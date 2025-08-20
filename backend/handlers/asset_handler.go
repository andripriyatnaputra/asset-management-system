package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

func CreateAsset(c *gin.Context) {
	var newAsset models.Asset
	if err := c.ShouldBindJSON(&newAsset); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Use a transaction to ensure asset and budget transaction are created together
	tx, err := database.Pool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	// 1. Create the asset
	assetQuery := `INSERT INTO assets (name, asset_tag, status, asset_type_id, purchase_date, initial_price) 
				   VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`
	err = tx.QueryRow(context.Background(), assetQuery,
		newAsset.Name, newAsset.AssetTag, newAsset.Status,
		newAsset.AssetTypeID, newAsset.PurchaseDate, newAsset.InitialPrice,
	).Scan(&newAsset.ID, &newAsset.CreatedAt, &newAsset.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset", "detail": err.Error()})
		return
	}

	// 2. Find an active budget for the purchase date
	var budgetID int64
	budgetQuery := `SELECT id FROM budgets WHERE start_date <= $1 AND end_date >= $1 AND deleted_at IS NULL LIMIT 1`
	err = tx.QueryRow(context.Background(), budgetQuery, newAsset.PurchaseDate).Scan(&budgetID)

	if err == nil { // If a budget is found
		// 3. Create a budget transaction
		txQuery := `INSERT INTO budget_transactions (budget_id, asset_id, amount, transaction_date)
					VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(context.Background(), txQuery, budgetID, newAsset.ID, newAsset.InitialPrice, newAsset.PurchaseDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create budget transaction", "detail": err.Error()})
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, newAsset)
}

type AssignRequest struct {
	EmployeeNIK string `json:"employee_nik" binding:"required"`
	Notes       string `json:"notes"`
}

func AssignAsset(c *gin.Context) {
	assetID := c.Param("id")
	var req AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
	tx, err := database.Pool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())
	var currentAssignmentID int
	err = tx.QueryRow(context.Background(), "SELECT id FROM asset_assignments WHERE asset_id = $1 AND returned_at IS NULL", assetID).Scan(&currentAssignmentID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Asset is already assigned"})
		return
	}
	var employeeID int64
	err = tx.QueryRow(context.Background(), "SELECT id FROM employees WHERE employee_nik = $1", req.EmployeeNIK).Scan(&employeeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	_, err = tx.Exec(context.Background(), "INSERT INTO asset_assignments (asset_id, employee_id, notes) VALUES ($1, $2, $3)", assetID, employeeID, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment record"})
		return
	}
	_, err = tx.Exec(context.Background(), "UPDATE assets SET status = 'Assigned' WHERE id = $1", assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update asset status"})
		return
	}
	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Asset assigned successfully"})
}

type ReturnRequest struct {
	NextStatus string `json:"next_status" binding:"required"` // e.g., "In Stock", "In Repair"
	Notes      string `json:"notes"`
}

// ReturnAsset menangani logika pengembalian sebuah aset
func ReturnAsset(c *gin.Context) {
	assetID := c.Param("id")
	var req ReturnRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	tx, err := database.Pool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	// 1. Cari assignment yang aktif untuk aset ini
	var assignmentID int64
	err = tx.QueryRow(context.Background(),
		"SELECT id FROM asset_assignments WHERE asset_id = $1 AND returned_at IS NULL",
		assetID).Scan(&assignmentID)

	if err != nil {
		// Jika tidak ada baris ditemukan (pgx.ErrNoRows), berarti aset tidak sedang di-assign
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset is not currently assigned or not found"})
		return
	}

	// 2. Update catatan assignment tersebut dengan menandai waktu pengembalian
	_, err = tx.Exec(context.Background(),
		"UPDATE asset_assignments SET returned_at = NOW(), notes = $1 WHERE id = $2",
		req.Notes, assignmentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment record"})
		return
	}

	// 3. Update status aset di tabel utama
	_, err = tx.Exec(context.Background(),
		"UPDATE assets SET status = $1 WHERE id = $2",
		req.NextStatus, assetID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update asset status"})
		return
	}

	// 4. Jika semua berhasil, commit transaksi
	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset returned successfully"})
}

func GetAssetHistory(c *gin.Context) {
	assetID := c.Param("id")
	var history []models.AssetHistoryResponse

	// Query ini menggabungkan (JOIN) tabel assignments dengan employees
	// untuk mendapatkan nama karyawan, bukan hanya ID-nya.
	// Diurutkan dari yang paling baru (DESC).
	query := `
		SELECT 
			aa.id,
			e.employee_nik,
			e.name,
			aa.assigned_at,
			aa.returned_at,
			aa.notes
		FROM 
			asset_assignments aa
		JOIN 
			employees e ON aa.employee_id = e.id
		WHERE 
			aa.asset_id = $1
		ORDER BY 
			aa.assigned_at DESC`

	rows, err := database.Pool.Query(context.Background(), query, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch asset history"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var record models.AssetHistoryResponse
		if err := rows.Scan(
			&record.AssignmentID,
			&record.EmployeeNIK,
			&record.EmployeeName,
			&record.AssignedAt,
			&record.ReturnedAt,
			&record.Notes,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan history record"})
			return
		}
		history = append(history, record)
	}

	// Jika tidak ada riwayat sama sekali, kembalikan array kosong, bukan error.
	if history == nil {
		history = make([]models.AssetHistoryResponse, 0)
	}

	c.JSON(http.StatusOK, history)
}

func GetAssetDepreciation(c *gin.Context) {
	assetID := c.Param("id")
	var asset models.Asset

	// 1. Ambil data aset dari database
	query := "SELECT name, asset_tag, purchase_date, initial_price FROM assets WHERE id = $1"
	err := database.Pool.QueryRow(context.Background(), query, assetID).Scan(&asset.Name, &asset.AssetTag, &asset.PurchaseDate, &asset.InitialPrice)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}

	// 2. Lakukan kalkulasi depresiasi
	usefulLifeYears := 3.0
	depreciationPerYear := asset.InitialPrice / usefulLifeYears

	// Hitung umur aset dalam tahun
	ageInDays := time.Since(asset.PurchaseDate).Hours() / 24
	ageInYears := ageInDays / 365.25

	// Hitung total depresiasi hingga saat ini
	totalDepreciation := depreciationPerYear * ageInYears
	if totalDepreciation > asset.InitialPrice {
		totalDepreciation = asset.InitialPrice // Nilai tidak bisa di bawah nol
	}

	// Hitung nilai buku saat ini
	currentBookValue := asset.InitialPrice - totalDepreciation

	// 3. Kembalikan hasil kalkulasi dalam format JSON yang informatif
	c.JSON(http.StatusOK, gin.H{
		"asset_name":            asset.Name,
		"asset_tag":             asset.AssetTag,
		"initial_price":         asset.InitialPrice,
		"purchase_date":         asset.PurchaseDate.Format("2006-01-02"),
		"useful_life_years":     usefulLifeYears,
		"age_in_years":          ageInYears,
		"depreciation_per_year": depreciationPerYear,
		"total_depreciation":    totalDepreciation,
		"current_book_value":    currentBookValue,
	})
}

func GetAllAssets(c *gin.Context) {
	// --- Paginasi ---
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// --- Filter dan Pencarian ---
	searchQuery := c.Query("q")
	assetTypeID := c.Query("type_id")

	// --- Sorting ---
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	allowedSortBy := map[string]bool{"name": true, "asset_tag": true, "status": true, "purchase_date": true, "initial_price": true, "created_at": true}
	if !allowedSortBy[sortBy] {
		sortBy = "created_at"
	}

	// --- Membangun Query SQL secara Dinamis ---
	baseQuery := `FROM assets a LEFT JOIN asset_types at ON a.asset_type_id = at.id`
	whereClause := " WHERE a.deleted_at IS NULL"
	params := []interface{}{}
	paramCount := 1

	if searchQuery != "" {
		whereClause += fmt.Sprintf(" AND a.name ILIKE $%d", paramCount)
		params = append(params, "%"+searchQuery+"%")
		paramCount++
	}
	if assetTypeID != "" {
		whereClause += fmt.Sprintf(" AND a.asset_type_id = $%d", paramCount)
		params = append(params, assetTypeID)
		paramCount++
	}

	// Query untuk menghitung total record
	countQuery := "SELECT COUNT(a.id) " + baseQuery + whereClause
	var totalRecords int64
	err := database.Pool.QueryRow(context.Background(), countQuery, params...).Scan(&totalRecords)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count assets"})
		return
	}

	// Query untuk mengambil data
	dataQuery := fmt.Sprintf(`
		SELECT a.id, a.name, a.asset_tag, a.status, a.asset_type_id, at.name as asset_type_name, 
			   a.purchase_date, a.initial_price, a.created_at, a.updated_at 
		%s %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		baseQuery, whereClause, sortBy, sortOrder, paramCount, paramCount+1)

	params = append(params, limit, offset)

	rows, err := database.Pool.Query(context.Background(), dataQuery, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assets", "detail": err.Error()})
		return
	}
	defer rows.Close()

	assets := []models.Asset{}
	for rows.Next() {
		var asset models.Asset
		// --- PERBAIKAN UTAMA ADA DI SINI ---
		// Pastikan 10 kolom di SELECT cocok dengan 10 variabel di Scan
		if err := rows.Scan(
			&asset.ID, &asset.Name, &asset.AssetTag, &asset.Status,
			&asset.AssetTypeID, &asset.AssetTypeName,
			&asset.PurchaseDate, &asset.InitialPrice,
			&asset.CreatedAt, &asset.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning asset row: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan asset data"})
			return
		}
		assets = append(assets, asset)
	}

	// --- Struktur Respons ---
	c.JSON(http.StatusOK, gin.H{
		"data": assets,
		"pagination": gin.H{
			"total_records": totalRecords,
			"current_page":  page,
			"page_size":     limit,
			"total_pages":   (totalRecords + int64(limit) - 1) / int64(limit),
		},
	})
}

func GetAssetByID(c *gin.Context) {
	assetID := c.Param("id")
	var asset models.Asset

	query := `
		SELECT 
			a.id, a.name, a.asset_tag, a.status, 
			a.asset_type_id, at.name as asset_type_name, 
			a.purchase_date, a.initial_price, a.created_at, a.updated_at 
		FROM assets a
		LEFT JOIN asset_types at ON a.asset_type_id = at.id
		WHERE a.id = $1`

	err := database.Pool.QueryRow(context.Background(), query, assetID).Scan(
		&asset.ID, &asset.Name, &asset.AssetTag, &asset.Status,
		&asset.AssetTypeID, &asset.AssetTypeName,
		&asset.PurchaseDate, &asset.InitialPrice,
		&asset.CreatedAt, &asset.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}

	c.JSON(http.StatusOK, asset)
}

// UpdateAsset memperbarui data aset yang ada
func UpdateAsset(c *gin.Context) {
	assetID := c.Param("id")
	var assetData models.Asset

	if err := c.ShouldBindJSON(&assetData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	query := `
		UPDATE assets 
		SET 
			name = $1, asset_tag = $2, status = $3, asset_type_id = $4, 
			purchase_date = $5, initial_price = $6, updated_at = NOW()
		WHERE id = $7`

	commandTag, err := database.Pool.Exec(context.Background(), query,
		assetData.Name, assetData.AssetTag, assetData.Status, assetData.AssetTypeID,
		assetData.PurchaseDate, assetData.InitialPrice, assetID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update asset", "detail": err.Error()})
		return
	}

	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset updated successfully"})
}

func DeleteAsset(c *gin.Context) {
	assetID := c.Param("id")

	query := `UPDATE assets SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	commandTag, err := database.Pool.Exec(context.Background(), query, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset"})
		return
	}

	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found or already deleted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset deleted successfully"})
}
