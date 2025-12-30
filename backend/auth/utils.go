package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// GetUserID mengambil user_id dari context JWT, aman untuk berbagai tipe (int, int64, float64)
func getUserID(c *gin.Context) (int64, error) {
	if uid, ok := c.Get("user_id"); ok {
		switch v := uid.(type) {
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		default:
			return 0, errors.New("invalid user_id type")
		}
	}
	return 0, errors.New("missing user_id")
}
