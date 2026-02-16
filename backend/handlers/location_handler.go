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
)

type LocationRow struct {
	ID          int64      `json:"id"`
	ParentID    *int64     `json:"parent_id,omitempty"`   // ✅ baru
	ParentName  *string    `json:"parent_name,omitempty"` // ✅ opsional (hasil join)
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
      l.id,
      l.parent_id,
      p.site AS parent_name,
      l.site,
      l.building,
      l.room,
      l.description,
      l.status,
      CONCAT(
        COALESCE(p.site || ' > ', ''),
        l.site,
        CASE WHEN l.building IS NOT NULL AND l.building <> '' THEN ' - ' || l.building ELSE '' END,
        CASE WHEN l.room IS NOT NULL AND l.room <> '' THEN ' - ' || l.room ELSE '' END
      ) AS display,
      l.created_at,
      l.updated_at
    FROM locations l
    LEFT JOIN locations p ON p.id = l.parent_id
    WHERE l.status = 'active'
    ORDER BY
      COALESCE(p.site, l.site),
      (l.parent_id IS NULL) DESC,  -- parent tampil dulu
      l.site, l.building NULLS LAST, l.room NULLS LAST
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
			&r.ParentID,
			&r.ParentName,
			&r.Site,
			&r.Building,
			&r.Room,
			&r.Description,
			&r.Status,
			&r.Display,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
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

	// sebelum insert: validasi parent jika diisi
	if req.ParentID != nil {
		var ok bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM locations WHERE id=$1 AND status='active')`,
			*req.ParentID,
		).Scan(&ok)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent location"})
			return
		}
	}

	// 🔹 Prevent duplicate site+building+room
	var exists bool
	// duplicate guard: tambahkan parent_id
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(
		SELECT 1 FROM locations
		WHERE COALESCE(parent_id,0)=COALESCE($1,0)
		AND site=$2
		AND COALESCE(building,'') = COALESCE($3,'')
		AND COALESCE(room,'') = COALESCE($4,'')
		AND status='active'
	)`,
		req.ParentID, req.Site, req.Building, req.Room,
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
		`INSERT INTO locations (parent_id, site, building, room, description, status, created_at, updated_at, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,'active',NOW(),NOW(),$6,$6)
		RETURNING id`,
		req.ParentID, req.Site, req.Building, req.Room, req.Description, userID,
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

	if req.ParentID != nil && int64(id) == *req.ParentID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parent_id cannot be self"})
		return
	}

	if req.ParentID != nil {
		var ok bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM locations WHERE id=$1 AND status='active')`,
			*req.ParentID,
		).Scan(&ok)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent location"})
			return
		}
	}

	// 🔹 Check duplicate ONLY if site/building/room changed
	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(
			SELECT 1 FROM locations
			WHERE COALESCE(parent_id,0)=COALESCE($1,0)
			AND site=$2
			AND COALESCE(building,'') = COALESCE($3,'')
			AND COALESCE(room,'') = COALESCE($4,'')
			AND id <> $5
			AND status='active'
		)`,
		req.ParentID, req.Site, req.Building, req.Room, id,
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
      SET parent_id=$1,
          site=$2, building=$3, room=$4, description=$5,
          status=$6,
          updated_at=NOW(), updated_by=$7
    WHERE id=$8`,
		req.ParentID, req.Site, req.Building, req.Room, req.Description,
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

	// blok jika punya child aktif
	var childCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM locations WHERE parent_id=$1 AND status='active'`,
		id,
	).Scan(&childCount)
	if childCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error":           "cannot delete location; it has child locations",
			"children_linked": childCount,
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
