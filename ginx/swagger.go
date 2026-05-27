package ginx

// SwaggerResponse is a non-generic response type for Swagger annotations.
// swaggo cannot parse Go generics, so this is used in handler docs.
type SwaggerResponse struct {
	Code int         `json:"code" example:"0"`
	Msg  string      `json:"msg" example:"Success"`
	Data interface{} `json:"data,omitempty"`
}

// SwaggerPageData is a non-generic pagination type for Swagger annotations.
type SwaggerPageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total" example:"100"`
	PageNum  int         `json:"pageNum" example:"1"`
	PageSize int         `json:"pageSize" example:"20"`
}
