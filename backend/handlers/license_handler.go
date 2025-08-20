// File: backend/handlers/license_handler.go
package handlers

import (
	"context"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// CreateLicense adds a new software license
func CreateLicense(c *gin.Context) {
	var license models.SoftwareLicense
	if err := c.ShouldBindJSON(&license); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	query := `INSERT INTO software_licenses (name, license_key, total_seats, purchase_date, expiration_date, cost)
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`
	err := database.Pool.QueryRow(context.Background(), query,
		license.Name, license.LicenseKey, license.TotalSeats, license.PurchaseDate, license.ExpirationDate, license.Cost,
	).Scan(&license.ID, &license.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create license", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, license)
}

// GetAllLicenses retrieves all software licenses
func GetAllLicenses(c *gin.Context) {
	// Note: For a real app, you would add pagination, search, and sort here just like for assets.
	// For now, we will keep it simple.
	var licenses []models.SoftwareLicense

	query := `SELECT id, name, license_key, total_seats, purchase_date, expiration_date, cost, created_at 
			  FROM software_licenses WHERE deleted_at IS NULL ORDER BY name ASC`

	rows, err := database.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch licenses"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var license models.SoftwareLicense
		if err := rows.Scan(&license.ID, &license.Name, &license.LicenseKey, &license.TotalSeats, &license.PurchaseDate, &license.ExpirationDate, &license.Cost, &license.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan license data"})
			return
		}
		licenses = append(licenses, license)
	}
	c.JSON(http.StatusOK, licenses)
}

// GetSoftwareForAsset retrieves all software installed on a specific asset
func GetSoftwareForAsset(c *gin.Context) {
	assetID := c.Param("id")
	var installedSoftware []models.InstalledSoftwareInfo

	query := `
		SELECT si.id, sl.id, sl.name, sl.license_key, si.installation_date
		FROM software_installations si
		JOIN software_licenses sl ON si.license_id = sl.id
		WHERE si.asset_id = $1
		ORDER BY sl.name ASC`

	rows, err := database.Pool.Query(context.Background(), query, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch installed software"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var info models.InstalledSoftwareInfo
		if err := rows.Scan(&info.InstallationID, &info.LicenseID, &info.LicenseName, &info.LicenseKey, &info.InstallationDate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan installed software data"})
			return
		}
		installedSoftware = append(installedSoftware, info)
	}
	c.JSON(http.StatusOK, installedSoftware)
}

// InstallSoftwareOnAsset links a software license to an asset
func InstallSoftwareOnAsset(c *gin.Context) {
	assetID := c.Param("id")
	var installation struct {
		LicenseID int64 `json:"license_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&installation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	query := `INSERT INTO software_installations (asset_id, license_id) VALUES ($1, $2) RETURNING id`
	var installationID int64
	err := database.Pool.QueryRow(context.Background(), query, assetID, installation.LicenseID).Scan(&installationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to install software", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Software installed successfully", "installation_id": installationID})
}

func GetLicenseByID(c *gin.Context) {
	licenseID := c.Param("id")
	var license models.SoftwareLicense
	query := `SELECT id, name, license_key, total_seats, purchase_date, expiration_date, cost, created_at 
			  FROM software_licenses WHERE id = $1 AND deleted_at IS NULL`
	err := database.Pool.QueryRow(context.Background(), query, licenseID).Scan(
		&license.ID, &license.Name, &license.LicenseKey, &license.TotalSeats,
		&license.PurchaseDate, &license.ExpirationDate, &license.Cost, &license.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
		return
	}
	c.JSON(http.StatusOK, license)
}

// UpdateLicense updates an existing software license
func UpdateLicense(c *gin.Context) {
	licenseID := c.Param("id")
	var license models.SoftwareLicense
	if err := c.ShouldBindJSON(&license); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	query := `UPDATE software_licenses SET 
				name = $1, license_key = $2, total_seats = $3, purchase_date = $4, 
				expiration_date = $5, cost = $6
			  WHERE id = $7 AND deleted_at IS NULL`
	commandTag, err := database.Pool.Exec(context.Background(), query,
		license.Name, license.LicenseKey, license.TotalSeats, license.PurchaseDate,
		license.ExpirationDate, license.Cost, licenseID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update license", "detail": err.Error()})
		return
	}
	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "License updated successfully"})
}
