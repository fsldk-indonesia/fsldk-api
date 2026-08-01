// Package dto memuat DTO umum yang dipakai lintas modul.
package dto

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListQuery merepresentasikan parameter query standar untuk endpoint list.
type ListQuery struct {
	Page   int
	Limit  int
	Search string
	Sort   string // contoh: "-createdDate" (descending), "newsTitle" (ascending)
}

// Offset mengembalikan offset SQL berdasarkan page & limit.
func (q ListQuery) Offset() int {
	if q.Page < 1 {
		return 0
	}
	return (q.Page - 1) * q.Limit
}

// OrderBy menghasilkan klausa ORDER BY yang aman berdasarkan whitelist kolom.
// allowed memetakan nama field yang boleh di-sort ke nama kolom database.
func (q ListQuery) OrderBy(allowed map[string]string, def string) string {
	field := strings.TrimSpace(q.Sort)
	dir := "ASC"
	if strings.HasPrefix(field, "-") {
		dir = "DESC"
		field = strings.TrimPrefix(field, "-")
	}
	col, ok := allowed[field]
	if !ok {
		return def
	}
	return col + " " + dir
}

// ParseListQuery membaca parameter list dari query string dengan nilai default.
func ParseListQuery(c *gin.Context) ListQuery {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return ListQuery{
		Page:   page,
		Limit:  limit,
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.DefaultQuery("sort", "-createdDate"),
	}
}
