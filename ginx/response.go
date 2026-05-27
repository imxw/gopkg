package ginx

import (
	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/errorx"
)

// PageData is a generic pagination response body.
type PageData[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	PageNum  int   `json:"pageNum"`
	PageSize int   `json:"pageSize"`
}

// Response is a generic API response body.
type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data,omitempty"`
}

// Success sends a successful JSON response with data.
func Success[T any](c *gin.Context, data T) {
	c.JSON(errorx.OK.Status, Response[T]{
		Code: errorx.OK.BusinessCode,
		Msg:  errorx.OK.Message,
		Data: data,
	})
}

// SuccessWithMessage sends a successful JSON response with a custom message.
func SuccessWithMessage[T any](c *gin.Context, data T, message string) {
	c.JSON(errorx.OK.Status, Response[T]{
		Code: errorx.OK.BusinessCode,
		Msg:  message,
		Data: data,
	})
}

// SuccessNoData sends a successful JSON response without data.
func SuccessNoData(c *gin.Context) {
	Success(c, struct{}{})
}

// PaginationSuccess sends a paginated success response.
func PaginationSuccess[T any](c *gin.Context, list []T, total int64, pageNum, pageSize int) {
	Success(c, PageData[T]{
		List:     list,
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
	})
}

// Error sends an error JSON response derived from err.
func Error(c *gin.Context, err error) {
	ex := errorx.FromError(err)
	c.JSON(ex.Status, Response[any]{
		Code: ex.BusinessCode,
		Msg:  ex.Message,
		Data: nil,
	})
}

// BadRequest sends a 400 error response.
func BadRequest(c *gin.Context, msg string) {
	Error(c, errorx.ErrInvalidArgument.WithMessage(msg))
}

// InternalError sends a 500 error response.
func InternalError(c *gin.Context, msg string) {
	Error(c, errorx.ErrInternal.WithMessage(msg))
}

// Unauthorized sends a 401 error response.
func Unauthorized(c *gin.Context, msg string) {
	Error(c, errorx.ErrUnauthorized.WithMessage(msg))
}

// TokenExpired sends a token-expired error response.
func TokenExpired(c *gin.Context, msg string) {
	err := errorx.ErrTokenExpired
	if msg != "" {
		err = err.WithMessage(msg)
	}
	Error(c, err)
}

// TokenInvalid sends a token-invalid error response.
func TokenInvalid(c *gin.Context, msg string) {
	err := errorx.ErrTokenInvalid
	if msg != "" {
		err = err.WithMessage(msg)
	}
	Error(c, err)
}

// Forbidden sends a 403 error response.
func Forbidden(c *gin.Context, msg string) {
	Error(c, errorx.ErrForbidden.WithMessage(msg))
}

// NotFound sends a 404 error response.
func NotFound(c *gin.Context, msg string) {
	Error(c, errorx.ErrNotFound.WithMessage(msg))
}
