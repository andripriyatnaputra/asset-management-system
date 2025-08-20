// File: backend/handlers/asset_type_handler.go
package handlers

import (
	"context"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// CreateAssetType creates a new asset type
func CreateAssetType(c *gin.Context) {
	var newType models.AssetType
	if err := c.ShouldBindJSON(&newType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query := `INSERT INTO asset_types (name) VALUES ($1) RETURNING id`
	err := database.Pool.QueryRow(context.Background(), query, newType.Name).Scan(&newType.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset type"})
		return
	}
	c.JSON(http.StatusCreated, newType)
}

// GetAllAssetTypes retrieves all asset types
func GetAllAssetTypes(c *gin.Context) {
	var types []models.AssetType
	rows, err := database.Pool.Query(context.Background(), "SELECT id, name FROM asset_types ORDER BY name ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch asset types"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t models.AssetType
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan asset type data"})
			return
		}
		types = append(types, t)
	}
	c.JSON(http.StatusOK, types)
}

func UpdateAssetType(c *gin.Context) {
	typeID := c.Param("id")
	var typeData models.AssetType

	if err := c.ShouldBindJSON(&typeData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	query := `UPDATE asset_types SET name = $1 WHERE id = $2`
	commandTag, err := database.Pool.Exec(context.Background(), query, typeData.Name, typeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update asset type"})
		return
	}

	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset type not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Asset type updated successfully"})
}

func DeleteAssetType(c *gin.Context) {
	id := c.Param("id")
	var count int
	_ = database.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM assets WHERE asset_type_id = $1", id).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak bisa menghapus, tipe aset masih digunakan."})
		return
	}
	_, err := database.Pool.Exec(context.Background(), "DELETE FROM asset_types WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus tipe aset."})
		return
	}
	c.Status(http.StatusNoContent)
}
