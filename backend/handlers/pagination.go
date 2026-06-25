package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

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
	return pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

type pagedResponse struct {
	Data  interface{} `json:"data"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}
