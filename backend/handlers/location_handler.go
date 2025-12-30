// File: backend/handlers/location_handler.go
package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type LocationRow struct {
	ID          int64      `json:"id"`
	Site        string     `json:"site"`
	Building    *string    `json:"building,omitempty"`
	Room        *string    `json:"room,omitempty"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Display     string     `json:"display"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// ============================================================
// 📋 GET ALL LOCATIONS (active only + deterministic order)
// ============================================================
func GetAllLocations(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT
			id,
			site,
			building,
			room,
			description,
			status,
			CONCAT(
				site,
				CASE WHEN building IS NOT NULL AND building <> '' THEN ' - ' || building ELSE '' END,
				CASE WHEN room IS NOT NULL AND room <> '' THEN ' - ' || room ELSE '' END
			) AS display,
			created_at,
			updated_at
		FROM locations
		WHERE status = 'active'
		ORDER BY site, building NULLS LAST, room NULLS LAST
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query locations"})
		return
	}
	defer rows.Close()

	var list []LocationRow
	for rows.Next() {
		var r LocationRow
		if err := rows.Scan(
			&r.ID,
			&r.Site,
			&r.Building,
			&r.Room,
			&r.Description,
			&r.Status,
			&r.Display,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
			if err == pgx.ErrNoRows {
				break
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error on locations"})
			return
		}
		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// 🆕 CREATE LOCATION (A++) — validasi duplicate + audit
// ============================================================
func CreateLocation(c *gin.Context) {
	var req LocationRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// 🔹 Site required
	if strings.TrimSpace(req.Site) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "site is required"})
		return
	}

	// 🔹 Prevent duplicate site+building+room
	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(
			SELECT 1 FROM locations 
			WHERE site=$1 
			  AND COALESCE(building,'') = COALESCE($2,'') 
			  AND COALESCE(room,'') = COALESCE($3,'')
			  AND status='active'
		)`,
		req.Site, req.Building, req.Room,
	).Scan(&exists)

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "location already exists"})
		return
	}

	var userID *int64
	if v, ok := c.Get("userID"); ok {
		u := v.(int64)
		userID = &u
	}

	err := database.Pool.QueryRow(context.Background(),
		`INSERT INTO locations (site, building, room, description, status, created_at, updated_at, created_by, updated_by)
		 VALUES ($1,$2,$3,$4,'active',NOW(),NOW(),$5,$5)
		 RETURNING id`,
		req.Site, req.Building, req.Room, req.Description, userID,
	).Scan(&req.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create location"})
		return
	}

	middleware.LogAction(c, "locations", req.ID, "CREATE", req)
	c.JSON(http.StatusCreated, req)
}

// ============================================================
// ✏️ UPDATE LOCATION (A++) — audit + duplicate guard
// ============================================================
func UpdateLocation(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req LocationRow

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// 🔹 Check duplicate ONLY if site/building/room changed
	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(
			SELECT 1 FROM locations
			 WHERE site=$1
			   AND COALESCE(building,'') = COALESCE($2,'')
			   AND COALESCE(room,'') = COALESCE($3,'')
			   AND id <> $4
			   AND status='active'
		)`,
		req.Site, req.Building, req.Room, id,
	).Scan(&exists)

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate location"})
		return
	}

	var userID *int64
	if v, ok := c.Get("userID"); ok {
		u := v.(int64)
		userID = &u
	}

	cmdTag, err := database.Pool.Exec(context.Background(),
		`UPDATE locations
        SET site=$1, building=$2, room=$3, description=$4,
            status=$5,
            updated_at=NOW(), updated_by=$6
      WHERE id=$7`,
		req.Site, req.Building, req.Room, req.Description,
		req.Status, userID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update location"})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "location not found or inactive"})
		return
	}

	middleware.LogAction(c, "locations", int64(id), "UPDATE", req)
	c.JSON(http.StatusOK, gin.H{"message": "location updated"})
}

// ============================================================
// 🗑 DELETE LOCATION — soft + asset linkage validation
// ============================================================
func DeleteLocation(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// 🔹 Check reference in assets
	var assetCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM assets WHERE location_id=$1 AND deleted_at IS NULL`,
		id,
	).Scan(&assetCount)

	if assetCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error":         "cannot delete location; assets still linked",
			"assets_linked": assetCount,
		})
		return
	}

	// 🔹 Soft delete
	var userID *int64
	if v, ok := c.Get("userID"); ok {
		u := v.(int64)
		userID = &u
	}

	_, err := database.Pool.Exec(context.Background(),
		`UPDATE locations
		    SET status='inactive', updated_at=NOW(), updated_by=$1
		  WHERE id=$2 AND status='active'`,
		userID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate location"})
		return
	}

	middleware.LogAction(c, "locations", int64(id), "DELETE", gin.H{"status": "inactive"})
	c.JSON(http.StatusOK, gin.H{"message": "location deactivated"})
}
