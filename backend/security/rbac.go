package security

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireRoles ensures the authenticated user's role is one of allowed roles.
// Usage: router.POST("/employees", RequireRoles("admin","super_admin"), handlers.CreateEmployee)
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, r := range roles {
		allowed[strings.ToLower(r)] = struct{}{}
	}

	return func(c *gin.Context) {
		rv, ok := c.Get("role")
		if !ok || rv == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		roleStr, ok := rv.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid role type"})
			return
		}
		if _, ok := allowed[strings.ToLower(roleStr)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
