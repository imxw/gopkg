package errorx

import (
	"net/http"
)

// 通用错误码
// 这些错误码用于所有模块共享的通用场景
// 范围: 0 (成功), 1000-1050 (通用业务), 2000-2099 (认证/Token)
const (
	CodeSuccess         = 0     // 成功
	CodeInvalidArgument = 1000  // 参数验证失败
	CodeUnauthorized    = 1001  // 未认证
	CodeForbidden       = 1003  // 权限不足
	CodeNotFound        = 1004  // 资源未找到
	CodeConflict        = 1009  // 操作冲突
	CodeTooManyRequests = 1029  // 请求过于频繁
	CodeInternal        = 1050  // 内部服务器错误
	CodeTokenExpired    = 2000  // Token已过期
	CodeTokenInvalid    = 2001  // Token无效
)

var (
	// OK 代表请求成功
	OK = &ErrorX{
		BusinessCode: CodeSuccess,
		HTTPCode:     http.StatusOK,
		Message:      "Success",
	}

	// ErrInternal 所有未知的服务器端错误
	ErrInternal = &ErrorX{
		BusinessCode: CodeInternal,
		HTTPCode:     http.StatusInternalServerError,
		Message:      "Internal server error",
	}

	// ErrNotFound 资源未找到
	ErrNotFound = &ErrorX{
		BusinessCode: CodeNotFound,
		HTTPCode:     http.StatusNotFound,
		Message:      "Resource not found",
	}

	// ErrInvalidArgument 参数验证失败
	ErrInvalidArgument = &ErrorX{
		BusinessCode: CodeInvalidArgument,
		HTTPCode:     http.StatusBadRequest,
		Message:      "Argument verification failed",
	}

	// ErrUnauthorized 认证失败
	ErrUnauthorized = &ErrorX{
		BusinessCode: CodeUnauthorized,
		HTTPCode:     http.StatusUnauthorized,
		Message:      "Unauthenticated",
	}

	// ErrForbidden 权限不足
	ErrForbidden = &ErrorX{
		BusinessCode: CodeForbidden,
		HTTPCode:     http.StatusForbidden,
		Message:      "Permission denied",
	}

	// ErrConflict 操作因业务冲突失败
	ErrConflict = &ErrorX{
		BusinessCode: CodeConflict,
		HTTPCode:     http.StatusConflict,
		Message:      "The requested operation has failed. Please try again later.",
	}

	// ErrTooManyRequests 操作频率超限
	ErrTooManyRequests = &ErrorX{
		BusinessCode: CodeTooManyRequests,
		HTTPCode:     http.StatusTooManyRequests,
		Message:      "Too many requests. Please try again later.",
	}

	// ErrTokenExpired Token已过期
	ErrTokenExpired = &ErrorX{
		BusinessCode: CodeTokenExpired,
		HTTPCode:     http.StatusUnauthorized,
		Message:      "Token expired, please refresh",
	}

	// ErrTokenInvalid Token无效
	ErrTokenInvalid = &ErrorX{
		BusinessCode: CodeTokenInvalid,
		HTTPCode:     http.StatusUnauthorized,
		Message:      "Token invalid, please re-authenticate",
	}
)
