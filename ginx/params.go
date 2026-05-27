package ginx

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PageParam holds pagination query parameters.
type PageParam struct {
	PageNum  int `form:"pageNum" json:"pageNum" binding:"omitempty,gte=1"`
	PageSize int `form:"pageSize" json:"pageSize" binding:"omitempty,gte=1,lte=100"`
}

// SortParam holds sorting query parameters.
type SortParam struct {
	Sort  string `form:"sort" json:"sort" binding:"omitempty,max=50"`
	Order string `form:"order" json:"order" binding:"omitempty,oneof=asc desc"`
}

// SafeOrder maps the user-supplied sort field to a safe database column name.
// Must be called in the Store layer; handler never concatenates SQL directly.
// allowed maps frontend field names to DB column names, e.g. {"hostname": "hostname", "createdAt": "created_at"}.
func (s SortParam) SafeOrder(allowed map[string]string) (column, order string, ok bool) {
	if s.Sort == "" {
		return "", "", false
	}
	column, ok = allowed[s.Sort]
	if !ok {
		return "", "", false
	}
	order = "ASC"
	if s.Order == "desc" {
		order = "DESC"
	}
	return column, order, true
}

// QueryParam is a common list-query parameter set covering most CRUD list endpoints.
type QueryParam struct {
	PageParam
	Keyword      string     `form:"keyword" json:"keyword" binding:"omitempty,max=128"`
	CreatedStart time.Time  `form:"createdStart" json:"createdStart" binding:"omitempty" time_format:"2006-01-02 15:04:05" time_utc:"false"`
	CreatedEnd   time.Time  `form:"createdEnd" json:"createdEnd" binding:"omitempty" time_format:"2006-01-02 15:04:05" time_utc:"false"`
	UpdatedStart time.Time  `form:"updatedStart" json:"updatedStart" binding:"omitempty" time_format:"2006-01-02 15:04:05" time_utc:"false"`
	UpdatedEnd   time.Time  `form:"updatedEnd" json:"updatedEnd" binding:"omitempty" time_format:"2006-01-02 15:04:05" time_utc:"false"`
	Status       *int8      `form:"status" json:"status" binding:"omitempty,oneof=0 1"`
	SortParam
}

// ToBaseQuery returns the common query fields for database queries.
func (q *QueryParam) ToBaseQuery() (keyword string, status *int8, start, end time.Time, sort, order string) {
	return q.Keyword, q.Status, q.CreatedStart, q.CreatedEnd, q.Sort, q.Order
}

// HasCreatedTimeRange reports whether a created-time range filter is present.
func (q *QueryParam) HasCreatedTimeRange() bool {
	return !q.CreatedStart.IsZero() || !q.CreatedEnd.IsZero()
}

// Sort order constants.
const (
	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// ParsePagination extracts pageNum and pageSize from query string.
// Returns clamped values: pageNum in [1, 10000], pageSize in [1, 100].
func ParsePagination(c *gin.Context, defaultPageNum, defaultPageSize int) (pageNum, pageSize int) {
	pageNum = atoiDefault(c.DefaultQuery("pageNum", ""), defaultPageNum)
	pageSize = atoiDefault(c.DefaultQuery("pageSize", ""), defaultPageSize)
	if pageNum < 1 {
		pageNum = defaultPageNum
	}
	if pageNum > 10000 {
		pageNum = 10000
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return pageNum, pageSize
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
