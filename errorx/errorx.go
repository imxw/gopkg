// Package errorx provides a structured error type with business error codes, HTTP status codes, and message for consistent API error handling.
package errorx

import (
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
)

// ErrorX 结构化业务错误。字段私有（v0.12 收敛）：导出可变字段等于让任何
// 代码能改写全局哨兵（OK/ErrInternal 等是包级共享指针）——不可变纪律只对
// Wrap/With* 副本族有效，对直接字段赋值无效。只读访问走访问器
// （Code/HTTPStatus/Name/Message/Extra/TraceID）或 Extract 投影。
type ErrorX struct {
	businessCode int
	status       int
	name         string
	message      string
	rawErr       error
	extra        map[string]any
	traceID      string
}

// New 创建 ErrorX 实例
func New(businessCode, status int, name, message string) *ErrorX {
	return &ErrorX{
		businessCode: businessCode,
		status:       status,
		name:         name,
		message:      message,
	}
}

func (e *ErrorX) Error() string {
	base := fmt.Sprintf("biz_code=%d status=%d msg=%s", e.businessCode, e.status, e.message)
	if e.rawErr != nil {
		base += fmt.Sprintf(" raw_err=%v", e.rawErr)
	}
	if len(e.extra) > 0 {
		// 键排序输出：map 迭代序不稳定，日志不可 diff（v0.12 修正）。
		keys := make([]string, 0, len(e.extra))
		for k := range e.extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf("%s=%v", k, e.extra[k]))
		}
		base += fmt.Sprintf(" extra=%s", strings.Join(pairs, ","))
	}
	return base
}

func (e *ErrorX) Unwrap() error {
	return e.rawErr
}

func (e *ErrorX) Is(target error) bool {
	t, ok := target.(*ErrorX)
	if !ok {
		return false
	}
	return e.businessCode == t.businessCode
}

// Code 业务码（int 分配表列；客户端语义分派走 Name）。
func (e *ErrorX) Code() int { return e.businessCode }

// HTTPStatus 对应的 HTTP 状态码。
func (e *ErrorX) HTTPStatus() int { return e.status }

// Name 稳定串码（CLIENT 按 error 列语义分派）。
func (e *ErrorX) Name() string { return e.name }

// Message 人类可读消息。
func (e *ErrorX) Message() string { return e.message }

// TraceID 显式携带的追踪 ID（ginx 错误腿 RequestID 列来源之一）。
func (e *ErrorX) TraceID() string { return e.traceID }

// Extra 只读投影：副本出（外部改写不得穿透回 ErrorX）。
func (e *ErrorX) Extra() map[string]any {
	if len(e.extra) == 0 {
		return nil
	}
	out := make(map[string]any, len(e.extra))
	maps.Copy(out, e.extra)
	return out
}

// Extract 将任意 error 投影为 (code, name, msg, ok)——WS 帧/gRPC/日志等
// 非 HTTP 出口的只读消费形态（ginx.Error 的信封之外，此前消费方手写
// errors.As 提取字段）。
func Extract(err error) (code int, name, msg string, ok bool) {
	ex, found := errors.AsType[*ErrorX](err)
	if !found {
		return 0, "", "", false
	}
	return ex.businessCode, ex.name, ex.message, true
}

// Wrap 返回副本并设置原始错误，不修改接收者
func (e *ErrorX) Wrap(rawErr error) *ErrorX {
	cp := *e
	cp.rawErr = rawErr
	return &cp
}

// WithMessage 返回副本并设置消息，不修改接收者
func (e *ErrorX) WithMessage(msg string) *ErrorX {
	cp := *e
	cp.message = msg
	return &cp
}

// WithExtra 返回副本并添加额外信息，不修改接收者
func (e *ErrorX) WithExtra(k string, v any) *ErrorX {
	cp := *e
	if cp.extra == nil {
		cp.extra = make(map[string]any)
	} else {
		newExtra := make(map[string]any, len(cp.extra)+1)
		maps.Copy(newExtra, cp.extra)
		cp.extra = newExtra
	}
	cp.extra[k] = v
	return &cp
}

// WithTraceID 返回副本并设置追踪ID，不修改接收者
func (e *ErrorX) WithTraceID(traceID string) *ErrorX {
	cp := *e
	cp.traceID = traceID
	return &cp
}

func (e *ErrorX) Data() map[string]any {
	// Extra 先落、信封键后写：WithExtra("code", …) 这类自定义键不得
	// 覆盖真实业务码/名称/消息——信封键是协议约定，优先级恒高于载荷。
	data := make(map[string]any, len(e.extra)+4)
	maps.Copy(data, e.extra)
	data["code"] = e.businessCode
	data["name"] = e.name
	data["msg"] = e.message
	data["data"] = struct{}{}
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
