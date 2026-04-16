package logger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestStringField(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"string field", "name", "test"},
		{"empty value", "key", ""},
		{"unicode", "name", "中文"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := StringField(tt.key, tt.value)
			assert.Equal(t, tt.key, f.Key)
		})
	}
}

func TestIntField(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value int
	}{
		{"positive", "count", 42},
		{"zero", "n", 0},
		{"negative", "temp", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := IntField(tt.key, tt.value)
			assert.Equal(t, tt.key, f.Key)
		})
	}
}

func TestInt64Field(t *testing.T) {
	f := Int64Field("big", int64(1<<62))
	assert.Equal(t, "big", f.Key)
}

func TestUint64Field(t *testing.T) {
	f := Uint64Field("uint", uint64(1<<63))
	assert.Equal(t, "uint", f.Key)
}

func TestBoolField(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := BoolField("flag", tt.value)
			assert.Equal(t, "flag", f.Key)
		})
	}
}

func TestFloat64Field(t *testing.T) {
	f := Float64Field("ratio", 3.14159)
	assert.Equal(t, "ratio", f.Key)
}

func TestTimeField(t *testing.T) {
	f := TimeField("created", time.Now())
	assert.Equal(t, "created", f.Key)
}

func TestErrorField(t *testing.T) {
	err := assert.AnError
	f := ErrorField(err)
	assert.Equal(t, "error", f.Key)
}

func TestAnyField(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"string", "hello"},
		{"int", 123},
		{"struct", struct{ X int }{1}},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := AnyField("data", tt.value)
			assert.Equal(t, "data", f.Key)
		})
	}
}

func TestFieldsReturnZapField(t *testing.T) {
	// Verify all field functions return zap.Field (not interface{})
	// by checking they are assignable to zap.Field
	var f zap.Field

	f = StringField("k", "v")
	assert.Equal(t, "k", f.Key)

	f = IntField("k", 1)
	assert.Equal(t, "k", f.Key)

	f = Int64Field("k", 1)
	assert.Equal(t, "k", f.Key)

	f = Uint64Field("k", 1)
	assert.Equal(t, "k", f.Key)

	f = BoolField("k", true)
	assert.Equal(t, "k", f.Key)

	f = Float64Field("k", 1.0)
	assert.Equal(t, "k", f.Key)

	f = ErrorField(assert.AnError)
	assert.Equal(t, "error", f.Key)

	f = AnyField("k", "v")
	assert.Equal(t, "k", f.Key)
}
