package logger

import (
	"time"

	"go.uber.org/zap"
)

// -------------------------- 字段构造方法（封装 zap，对外无感知） --------------------------
// StringField 构造字符串类型字段
func StringField(key, value string) zap.Field {
	return zap.String(key, value)
}

// ErrorField 构造错误类型字段
func ErrorField(err error) zap.Field {
	return zap.Error(err)
}

// IntField 构造 int 类型字段
func IntField(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Int64Field 构造 int64 类型字段
func Int64Field(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

// Uint64Field 构造 uint64 类型字段
func Uint64Field(key string, value uint64) zap.Field {
	return zap.Uint64(key, value)
}

// BoolField 构造布尔类型字段
func BoolField(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

// Float64Field 构造浮点类型字段
func Float64Field(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

// TimeField 构造时间类型字段
func TimeField(key string, value time.Time) zap.Field {
	return zap.Time(key, value)
}

// AnyField 构造任意类型字段（兜底用，尽量少用）
func AnyField(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}
