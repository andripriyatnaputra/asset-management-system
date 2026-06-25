package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// GetAssetQRData mengembalikan data yang dibutuhkan frontend untuk render QR code.
func GetAssetQRData(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var asset struct {
		ID       int64
		Name     string
		AssetTag *string
	}
	err := database.Pool.QueryRow(c, `
		SELECT id, name, asset_tag FROM assets WHERE id = $1 AND deleted_at IS NULL
	`, assetID).Scan(&asset.ID, &asset.Name, &asset.AssetTag)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "aset tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	qrData := fmt.Sprintf("%s/assets/%d", baseURL, assetID)

	// Ambil QR code terakhir yang dibuat jika ada
	var existing models.AssetQRCode
	err = database.Pool.QueryRow(c, `
		SELECT id, asset_id, qr_data, format, label_data, printed_at, printed_by, created_by, created_at
		FROM asset_qr_codes WHERE asset_id = $1 ORDER BY created_at DESC LIMIT 1
	`, assetID).Scan(
		&existing.ID, &existing.AssetID, &existing.QRData, &existing.Format,
		&existing.LabelData, &existing.PrintedAt, &existing.PrintedBy,
		&existing.CreatedBy, &existing.CreatedAt,
	)

	response := gin.H{
		"asset_id":  assetID,
		"asset_name": asset.Name,
		"asset_tag":  asset.AssetTag,
		"qr_data":   qrData,
		"format":    "qr",
	}
	if err == nil {
		response["existing_code"] = existing
	}
	c.JSON(http.StatusOK, response)
}

// GenerateAssetQRCode menyimpan entry QR code ke DB (data untuk cetak label).
func GenerateAssetQRCode(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		Format    string  `json:"format"` // qr|barcode|datamatrix
		LabelData *string `json:"label_data"` // JSON string tambahan untuk label fisik
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Format == "" {
		req.Format = "qr"
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	qrData := fmt.Sprintf("%s/assets/%d", baseURL, assetID)

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO asset_qr_codes (asset_id, qr_data, format, label_data, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, assetID, qrData, req.Format, req.LabelData, actor).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"qr_data":  qrData,
		"format":   req.Format,
		"message":  "QR code entry berhasil dibuat",
	})
}

// LogQRPrint mencatat bahwa QR code sudah dicetak secara fisik.
func LogQRPrint(c *gin.Context) {
	assetID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	now := time.Now()
	_, err := database.Pool.Exec(c, `
		UPDATE asset_qr_codes SET
			printed_at = $1, printed_by = $2
		WHERE asset_id = $3
		  AND id = (SELECT id FROM asset_qr_codes WHERE asset_id = $3 ORDER BY created_at DESC LIMIT 1)
	`, now, actor, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "print log berhasil dicatat", "printed_at": now})
}

// LookupByQRData mencari aset berdasarkan qr_data (URL scan).
func LookupByQRData(c *gin.Context) {
	qrData := c.Query("data")
	if qrData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parameter 'data' diperlukan"})
		return
	}

	var assetID int64
	err := database.Pool.QueryRow(c, `
		SELECT asset_id FROM asset_qr_codes WHERE qr_data = $1 ORDER BY created_at DESC LIMIT 1
	`, qrData).Scan(&assetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QR data tidak ditemukan"})
		return
	}
	// Redirect ke detail aset
	c.JSON(http.StatusOK, gin.H{"asset_id": assetID, "redirect": fmt.Sprintf("/assets/%d", assetID)})
}
