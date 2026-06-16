// Package pagination centralizes page/page_size parsing and offset math so every
// service paginates list endpoints the same way (page is 1-indexed, size capped).
package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Parse extracts page/page_size from query params with safe defaults
// (page=1, page_size=20, capped at 100). Invalid values fall back to defaults.
func Parse(c *gin.Context) (page, pageSize int) {
	page = defaultPage
	pageSize = defaultPageSize
	if n, err := strconv.Atoi(c.Query("page")); err == nil && n > 0 {
		page = n
	}
	if n, err := strconv.Atoi(c.Query("page_size")); err == nil && n > 0 {
		if n > maxPageSize {
			n = maxPageSize
		}
		pageSize = n
	}
	return
}

// Offset converts a 1-indexed page + size into a SQL OFFSET.
func Offset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}
