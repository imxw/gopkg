package ginx

import (
	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/contextx"
	"github.com/imxw/gopkg/errorx"
	"github.com/imxw/gopkg/logger"
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
	// 错误腿专列：Error 是 errorx NAME（稳定串码，供客户端语义分派——
	// int 业务码属服务端内部分配表，不作为客户端分支依据）；RequestID
	// 是请求关联 ID（对账日志/排障，对应 RFC 9457 的 instance）；Details
	// 是结构化错误载荷（校验报告/部分失败明细，ErrorWithDetails 填充）。
	// 仅错误腿填充；成功腿零值经 omitempty 不出线——向后兼容纯增量。
	Error     string `json:"error,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Details   any    `json:"details,omitempty"`
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
	if list == nil {
		list = []T{}
	}
	Success(c, PageData[T]{
		List:     list,
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
	})
}

// Error sends an error JSON response derived from err.
// If err is nil, it defaults to ErrInternal.
// 5xx errors are logged at ERROR level (server issues).
// 4xx errors are not logged (client issues, avoid log flooding).
func Error(c *gin.Context, err error) {
	writeError(c, err, nil)
}

// ErrorWithDetails behaves like Error and additionally attaches a structured
// payload (validation reports, partial-failure lists) to the error leg as the
// "details" field. err 仍按 Error 全语义处理（状态码/NAME/日志/requestId）。
func ErrorWithDetails(c *gin.Context, err error, details any) {
	writeError(c, err, details)
}

func writeError(c *gin.Context, err error, details any) {
	if err == nil {
		err = errorx.ErrInternal
	}
	ex := errorx.FromError(err)
	if ex.Status >= 500 {
		logger.CtxErrorw(c.Request.Context(), "server error",
			logger.IntField("biz_code", ex.BusinessCode), logger.ErrorField(err))
	}
	// 请求关联 ID：ErrorX 显式携带的 TraceID 优先，否则取 RequestID
	// 中间件注入的上下文值（NewEngine 默认挂链首）。
	requestID := ex.TraceID
	if requestID == "" {
		requestID = contextx.RequestID(c.Request.Context())
	}
	c.JSON(ex.Status, Response[any]{
		Code:      ex.BusinessCode,
		Msg:       ex.Message,
		Error:     ex.Name,
		RequestID: requestID,
		Data:      nil,
		Details:   details,
	})
}

// BadRequest sends a 400 error response.
func BadRequest(c *gin.Context, msg string) {
	Error(c, withFallbackMessage(errorx.ErrInvalidArgument, msg))
}

// InternalError sends a 500 error response.
func InternalError(c *gin.Context, msg string) {
	Error(c, withFallbackMessage(errorx.ErrInternal, msg))
}

// Unauthorized sends a 401 error response.
func Unauthorized(c *gin.Context, msg string) {
	Error(c, withFallbackMessage(errorx.ErrUnauthorized, msg))
}

// TokenExpired sends a token-expired error response.
func TokenExpired(c *gin.Context, msg string) {
	Error(c, withFallbackMessage(errorx.ErrTokenExpired, msg))
}

// TokenInvalid sends a token-invalid error response.
func TokenInvalid(c *gin.Context, msg string) {
	Error(c, withFallbackMessage(errorx.ErrTokenInvalid, msg))
}

// Forbidden sends a 403 error response.
func Forbidden(c *gin.Context, msg string) {
	Error(c, withFallbackMessage(errorx.ErrForbidden, msg))
}

// NotFound sends a 404 error response.
func NotFound(c *gin.Context, msg string) {
	Error(c, withFallbackMessage(errorx.ErrNotFound, msg))
}

// withFallbackMessage 统一便捷族语义：空 msg 保留哨兵默认消息（此前
// TokenExpired/TokenInvalid 有此守卫而其余便捷函数会把 Message 覆盖为
// 空串发出——同族两套行为分裂）。
func withFallbackMessage(sentinel *errorx.ErrorX, msg string) *errorx.ErrorX {
	if msg == "" {
		return sentinel
	}
	return sentinel.WithMessage(msg)
}
