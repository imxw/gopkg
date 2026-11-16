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
	CodeContentTooLarge = 1013 // 请求体超限
	CodeInternal        = 1050 // 内部服务器错误
	CodeTokenExpired    = 2000 // Token已过期
	CodeTokenInvalid    = 2001 // Token无效
)

var (
	// OK 代表请求成功
	OK = &ErrorX{
		businessCode: CodeSuccess,
		status:       http.StatusOK,
		name:         "SUCCESS",
		message:      "Success",
	}

	// ErrInternal 所有未知的服务器端错误
	ErrInternal = &ErrorX{
		businessCode: CodeInternal,
		status:       http.StatusInternalServerError,
		name:         "INTERNAL_ERROR",
		message:      "Internal server error",
	}

	// ErrNotFound 资源未找到
	ErrNotFound = &ErrorX{
		businessCode: CodeNotFound,
		status:       http.StatusNotFound,
		name:         "NOT_FOUND",
		message:      "Resource not found",
	}

	// ErrInvalidArgument 参数验证失败
	ErrInvalidArgument = &ErrorX{
		businessCode: CodeInvalidArgument,
		status:       http.StatusBadRequest,
		name:         "INVALID_ARGUMENT",
		message:      "Argument verification failed",
	}

	// ErrUnauthorized 认证失败
	ErrUnauthorized = &ErrorX{
		businessCode: CodeUnauthorized,
		status:       http.StatusUnauthorized,
		name:         "UNAUTHORIZED",
		message:      "Unauthenticated",
	}

	// ErrForbidden 权限不足
	ErrForbidden = &ErrorX{
		businessCode: CodeForbidden,
		status:       http.StatusForbidden,
		name:         "FORBIDDEN",
		message:      "Permission denied",
	}

	// ErrConflict 操作因业务冲突失败
	ErrConflict = &ErrorX{
		businessCode: CodeConflict,
		status:       http.StatusConflict,
		name:         "CONFLICT",
		message:      "The requested operation has failed. Please try again later.",
	}

	// ErrTooManyRequests 操作频率超限
	ErrTooManyRequests = &ErrorX{
		businessCode: CodeTooManyRequests,
		status:       http.StatusTooManyRequests,
		name:         "TOO_MANY_REQUESTS",
		message:      "Too many requests. Please try again later.",
	}

	// ErrContentTooLarge 请求体超限（HTTP 413 语义——此前 BodyLimit 借用
	// ErrInvalidArgument 出 400，与 RFC 9110 语义相悖）
	ErrContentTooLarge = &ErrorX{
		businessCode: CodeContentTooLarge,
		status:       http.StatusRequestEntityTooLarge,
		name:         "CONTENT_TOO_LARGE",
		message:      "Request body exceeds the allowed size.",
	}

	// ErrTokenExpired Token已过期
	ErrTokenExpired = &ErrorX{
		businessCode: CodeTokenExpired,
		status:       http.StatusUnauthorized,
		name:         "TOKEN_EXPIRED",
		message:      "Token expired, please refresh",
	}

	// ErrTokenInvalid Token无效
	ErrTokenInvalid = &ErrorX{
		businessCode: CodeTokenInvalid,
		status:       http.StatusUnauthorized,
		name:         "TOKEN_INVALID",
		message:      "Token invalid, please re-authenticate",
	}
)
