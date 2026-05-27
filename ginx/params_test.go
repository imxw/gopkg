package ginx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func parseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", s)
	return t
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		defaultPageNum  int
		defaultPageSize int
		wantPageNum     int
		wantPageSize    int
	}{
		{"defaults", "", 1, 20, 1, 20},
		{"valid values", "pageNum=3&pageSize=50", 1, 20, 3, 50},
		{"negative page", "pageNum=-1", 1, 20, 1, 20},
		{"zero page", "pageNum=0", 1, 20, 1, 20},
		{"oversized page", "pageNum=99999", 1, 20, 10000, 20},
		{"oversized pageSize", "pageSize=500", 1, 20, 1, 100},
		{"negative pageSize", "pageSize=-5", 1, 20, 1, 20},
		{"non-numeric", "pageNum=abc", 1, 20, 1, 20},
		{"exact max page", "pageNum=10000", 1, 20, 10000, 20},
		{"exact max pageSize", "pageSize=100", 1, 20, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			pageNum, pageSize := ParsePagination(c, tt.defaultPageNum, tt.defaultPageSize)
			assert.Equal(t, tt.wantPageNum, pageNum)
			assert.Equal(t, tt.wantPageSize, pageSize)
		})
	}
}

func TestSafeOrder(t *testing.T) {
	allowed := map[string]string{
		"hostname":   "hostname",
		"createdAt":  "created_at",
		"updatedAt":  "updated_at",
	}

	tests := []struct {
		name       string
		sort       string
		order      string
		wantCol    string
		wantOrder  string
		wantOK     bool
	}{
		{"empty sort", "", "", "", "", false},
		{"unknown field", "foobar", "", "", "", false},
		{"valid asc", "hostname", "asc", "hostname", "ASC", true},
		{"valid desc", "createdAt", "desc", "created_at", "DESC", true},
		{"default order", "updatedAt", "", "updated_at", "ASC", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SortParam{Sort: tt.sort, Order: tt.order}
			col, order, ok := s.SafeOrder(allowed)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantCol, col)
			assert.Equal(t, tt.wantOrder, order)
		})
	}
}

func TestHasCreatedTimeRange(t *testing.T) {
	q := &QueryParam{}
	assert.False(t, q.HasCreatedTimeRange())

	q.CreatedStart = parseTime("2026-01-01 00:00:00")
	assert.True(t, q.HasCreatedTimeRange())
}

func TestToBaseQuery(t *testing.T) {
	q := &QueryParam{
		Keyword: "test",
		SortParam: SortParam{Sort: "hostname", Order: "asc"},
	}
	keyword, status, _, _, sort, order := q.ToBaseQuery()
	assert.Equal(t, "test", keyword)
	assert.Nil(t, status)
	assert.Equal(t, "hostname", sort)
	assert.Equal(t, "asc", order)
}
