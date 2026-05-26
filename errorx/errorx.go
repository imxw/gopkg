// Package errorx provides a structured error type with business error codes, HTTP status codes, and message for consistent API error handling.
package errorx

import (
	"errors"
	"fmt"
	"maps"
)

// ErrorX 通用错误结构体
type ErrorX struct {
	BusinessCode int            `json:"-"`
	Status       int            `json:"-"`
	Name         string         `json:"-"`
	Message      string         `json:"-"`
	RawErr       error          `json:"-"`
	Extra        map[string]any `json:"-"`
	TraceID      string         `json:"-"`
}

// New 创建 ErrorX 实例
func New(businessCode, status int, name, message string) *ErrorX {
	return &ErrorX{
		BusinessCode: businessCode,
		Status:       status,
		Name:         name,
		Message:      message,
	}
}

func (e *ErrorX) Error() string {
	base := fmt.Sprintf("biz_code=%d status=%d msg=%s", e.BusinessCode, e.Status, e.Message)
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

func (e *ErrorX) Is(target error) bool {
	t, ok := target.(*ErrorX)
	if !ok {
		return false
	}
	return e.BusinessCode == t.BusinessCode
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
		newExtra := make(map[string]any, len(cp.Extra)+1)
		maps.Copy(newExtra, cp.Extra)
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

func (e *ErrorX) Data() map[string]any {
	data := map[string]any{
		"code": e.BusinessCode,
		"name": e.Name,
		"msg":  e.Message,
		"data": struct{}{},
	}
	maps.Copy(data, e.Extra)
	return data
}

// FromError 将任意 error 解析为 ErrorX
func FromError(err error) *ErrorX {
	if err == nil {
		return nil
	}

	ex, ok := errors.AsType[*ErrorX](err)
	if ok {
		return ex
	}

	return ErrInternal.Wrap(err)
}
