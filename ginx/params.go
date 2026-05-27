package ginx

const (
	MaxPageNum  = 10000
	MaxPageSize = 100
)

// PageParam holds pagination query parameters.
type PageParam struct {
	PageNum  int `form:"pageNum" json:"pageNum" binding:"omitempty,gte=1"`
	PageSize int `form:"pageSize" json:"pageSize" binding:"omitempty,gte=1,lte=100"`
}

// Normalize fills missing values with defaults (pageNum=1, pageSize=20)
// and clamps to allowed ranges.
func (p *PageParam) Normalize() {
	if p.PageNum < 1 {
		p.PageNum = 1
	}
	if p.PageNum > MaxPageNum {
		p.PageNum = MaxPageNum
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
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
