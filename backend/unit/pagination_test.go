// Package unit contains pure unit tests with no database dependency.
package unit

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// pagination mirrors handlers.pagination to allow testing without importing the handlers package
// (which would trigger TestMain and require a database connection).
type pagination struct {
	Page   int
	Limit  int
	Offset int
}

func getPagination(c *gin.Context) pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return pagination{Page: page, Limit: limit, Offset: (page - 1) * limit}
}

func ginCtx(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/?"+query, nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

func TestGetPagination_Defaults(t *testing.T) {
	pg := getPagination(ginCtx(""))
	assert.Equal(t, 1, pg.Page)
	assert.Equal(t, 20, pg.Limit)
	assert.Equal(t, 0, pg.Offset)
}

func TestGetPagination_CustomValues(t *testing.T) {
	pg := getPagination(ginCtx("page=3&limit=10"))
	assert.Equal(t, 3, pg.Page)
	assert.Equal(t, 10, pg.Limit)
	assert.Equal(t, 20, pg.Offset) // (3-1)*10
}

func TestGetPagination_ClampPageMin(t *testing.T) {
	pg := getPagination(ginCtx("page=0"))
	assert.Equal(t, 1, pg.Page)
	assert.Equal(t, 0, pg.Offset)
}

func TestGetPagination_ClampLimitMin(t *testing.T) {
	pg := getPagination(ginCtx("limit=0"))
	assert.Equal(t, 20, pg.Limit)
}

func TestGetPagination_ClampLimitMax(t *testing.T) {
	pg := getPagination(ginCtx("limit=999"))
	assert.Equal(t, 100, pg.Limit)
}

func TestGetPagination_NegativePage(t *testing.T) {
	pg := getPagination(ginCtx("page=-5"))
	assert.Equal(t, 1, pg.Page)
	assert.Equal(t, 0, pg.Offset)
}

func TestGetPagination_OffsetFormula(t *testing.T) {
	pg := getPagination(ginCtx("page=5&limit=25"))
	assert.Equal(t, 5, pg.Page)
	assert.Equal(t, 25, pg.Limit)
	assert.Equal(t, 100, pg.Offset) // (5-1)*25
}
