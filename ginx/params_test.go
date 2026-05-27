package ginx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageParam_Normalize(t *testing.T) {
	tests := []struct {
		name         string
		pageNum      int
		pageSize     int
		wantPageNum  int
		wantPageSize int
	}{
		{"defaults", 0, 0, 1, 20},
		{"valid values", 3, 50, 3, 50},
		{"negative page", -1, 0, 1, 20},
		{"oversized page", 99999, 0, MaxPageNum, 20},
		{"oversized pageSize", 0, 500, 1, MaxPageSize},
		{"negative pageSize", 0, -5, 1, 20},
		{"exact max page", MaxPageNum, 0, MaxPageNum, 20},
		{"exact max pageSize", 0, MaxPageSize, 1, MaxPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PageParam{PageNum: tt.pageNum, PageSize: tt.pageSize}
			p.Normalize()
			assert.Equal(t, tt.wantPageNum, p.PageNum)
			assert.Equal(t, tt.wantPageSize, p.PageSize)
		})
	}
}

func TestSafeOrder(t *testing.T) {
	allowed := map[string]string{
		"hostname":  "hostname",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
	}

	tests := []struct {
		name      string
		sort      string
		order     string
		wantCol   string
		wantOrder string
		wantOK    bool
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
