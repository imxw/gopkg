package errorx

import (
	"net/http"
)

// 通用错误码
// 这些错误码用于所有模块共享的通用场景
// 范围: 0 (成功), 1000-1050 (通用业务), 2000-2099 (认证/Token)
//
// 码段分配约定（消费方自定义业务码必须避开本段）：1000-1050 通用 /
// 2000-2099 认证 Token 为 gopkg 保留段；消费方从自己的段位起分配
// （如 vortex 用 50xx/53xx/54xx），并在消费方侧登记码段分配表 +
// 守卫测试防撞——gopkg 不感知消费方段位，撞码无库级防线。
const (
	CodeSuccess         = 0    // 成功
	CodeInvalidArgument = 1000 // 参数验证失败
	CodeUnauthorized    = 1001 // 未认证
	CodeForbidden       = 1003 // 权限不足
	CodeNotFound        = 1004 // 资源未找到
	CodeConflict        = 1009 // 操作冲突
	CodeTooManyRequests = 1029 // 请求过于频繁
	CodeInternal        = 1050 // 内部服务器错误
	CodeTokenExpired    = 2000 // Token已过期
	CodeTokenInvalid    = 2001 // Token无效
)

var (
	// OK 代表请求成功
	OK = &ErrorX{
		BusinessCode: CodeSuccess,
		Status:       http.StatusOK,
		Name:         "SUCCESS",
		Message:      "Success",
	}

	// ErrInternal 所有未知的服务器端错误
	ErrInternal = &ErrorX{
		BusinessCode: CodeInternal,
		Status:       http.StatusInternalServerError,
		Name:         "INTERNAL_ERROR",
		Message:      "Internal server error",
	}

	// ErrNotFound 资源未找到
	ErrNotFound = &ErrorX{
		BusinessCode: CodeNotFound,
		Status:       http.StatusNotFound,
		Name:         "NOT_FOUND",
		Message:      "Resource not found",
	}

	// ErrInvalidArgument 参数验证失败
	ErrInvalidArgument = &ErrorX{
		BusinessCode: CodeInvalidArgument,
		Status:       http.StatusBadRequest,
		Name:         "INVALID_ARGUMENT",
		Message:      "Argument verification failed",
	}

	// ErrUnauthorized 认证失败
	ErrUnauthorized = &ErrorX{
		BusinessCode: CodeUnauthorized,
		Status:       http.StatusUnauthorized,
		Name:         "UNAUTHORIZED",
		Message:      "Unauthenticated",
	}

	// ErrForbidden 权限不足
	ErrForbidden = &ErrorX{
		BusinessCode: CodeForbidden,
		Status:       http.StatusForbidden,
		Name:         "FORBIDDEN",
		Message:      "Permission denied",
	}

	// ErrConflict 操作因业务冲突失败
	ErrConflict = &ErrorX{
		BusinessCode: CodeConflict,
		Status:       http.StatusConflict,
		Name:         "CONFLICT",
		Message:      "The requested operation has failed. Please try again later.",
	}

	// ErrTooManyRequests 操作频率超限
	ErrTooManyRequests = &ErrorX{
		BusinessCode: CodeTooManyRequests,
		Status:       http.StatusTooManyRequests,
		Name:         "TOO_MANY_REQUESTS",
		Message:      "Too many requests. Please try again later.",
	}

	// ErrTokenExpired Token已过期
	ErrTokenExpired = &ErrorX{
		BusinessCode: CodeTokenExpired,
		Status:       http.StatusUnauthorized,
		Name:         "TOKEN_EXPIRED",
		Message:      "Token expired, please refresh",
	}

	// ErrTokenInvalid Token无效
	ErrTokenInvalid = &ErrorX{
		BusinessCode: CodeTokenInvalid,
		Status:       http.StatusUnauthorized,
		Name:         "TOKEN_INVALID",
		Message:      "Token invalid, please re-authenticate",
	}
)
