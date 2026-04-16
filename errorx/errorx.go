// Package errorx provides a structured error type with business error codes, HTTP status codes, and message for consistent API error handling.
package errorx

import (
	"errors"
	"fmt"
)

// ErrorX 通用错误结构体
type ErrorX struct {
	BusinessCode int            `json:"-"`
	HTTPCode     int            `json:"-"`
	Message      string         `json:"-"`
	RawErr       error          `json:"-"`
	Extra        map[string]any `json:"-"`
	TraceID      string         `json:"-"`
}

// New 创建 ErrorX 实例
func New(businessCode, httpCode int, message string) *ErrorX {
	return &ErrorX{
		BusinessCode: businessCode,
		HTTPCode:     httpCode,
		Message:      message,
	}
}

func (e *ErrorX) Error() string {
	base := fmt.Sprintf("biz_code=%d http_code=%d msg=%s", e.BusinessCode, e.HTTPCode, e.Message)
	if e.RawErr != nil {
		base += fmt.Sprintf(" raw_err=%v", e.RawErr)
	}
	if len(e.Extra) > 0 {
		base += fmt.Sprintf(" extra=%v", e.Extra)
	}
	return base
}

func (e *ErrorX) Unwrap() error {
	return e.RawErr
}

// Wrap 返回副本并设置原始错误，不修改接收者
func (e *ErrorX) Wrap(rawErr error) *ErrorX {
	cp := *e
	cp.RawErr = rawErr
	return &cp
}

// WithMessage 返回副本并设置消息，不修改接收者
func (e *ErrorX) WithMessage(msg string) *ErrorX {
	cp := *e
	cp.Message = msg
	return &cp
}

// WithExtra 返回副本并添加额外信息，不修改接收者
func (e *ErrorX) WithExtra(k string, v any) *ErrorX {
	cp := *e
	if cp.Extra == nil {
		cp.Extra = make(map[string]any)
	} else {
		// 复制 map 避免共享引用
		newExtra := make(map[string]any, len(cp.Extra)+1)
		for ek, ev := range cp.Extra {
			newExtra[ek] = ev
		}
		cp.Extra = newExtra
	}
	cp.Extra[k] = v
	return &cp
}

// WithTraceID 返回副本并设置追踪ID，不修改接收者
func (e *ErrorX) WithTraceID(traceID string) *ErrorX {
	cp := *e
	cp.TraceID = traceID
	return &cp
}

func (e *ErrorX) Status() int {
	return e.HTTPCode
}

func (e *ErrorX) Data() map[string]interface{} {
	data := map[string]interface{}{
		"code": e.BusinessCode,
		"msg":  e.Message,
		"data": struct{}{},
	}
	for k, v := range e.Extra {
		data[k] = v
	}
	return data
}

// FromError 将任意 error 解析为 ErrorX
func FromError(err error) *ErrorX {
	if err == nil {
		return nil
	}

	var ex *ErrorX
	if errors.As(err, &ex) {
		return ex
	}

	return ErrInternal.Wrap(err)
}
