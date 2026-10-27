// Package pgxutil provides type conversion helpers between Go types and pgx/pgtype.
package pgxutil

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ToPgText converts a string to pgtype.Text.
func ToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// ToPgTextPtr converts a *string to pgtype.Text.
func ToPgTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// FromPgText converts pgtype.Text to string.
func FromPgText(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

// ToPgBool converts bool to pgtype.Bool.
func ToPgBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

// FromPgBool converts pgtype.Bool to bool.
func FromPgBool(v pgtype.Bool) bool {
	if !v.Valid {
		return false
	}
	return v.Bool
}

// ToPgInt4 converts int32 to pgtype.Int4.
func ToPgInt4(i int32) pgtype.Int4 {
	return pgtype.Int4{Int32: i, Valid: true}
}

// FromPgInt4 converts pgtype.Int4 to int32.
func FromPgInt4(v pgtype.Int4) int32 {
	if !v.Valid {
		return 0
	}
	return v.Int32
}

// ToPgInt4Ptr converts *int32 to pgtype.Int4.
func ToPgInt4Ptr(i *int32) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *i, Valid: true}
}

// ToPgBoolPtr converts *bool to pgtype.Bool.
func ToPgBoolPtr(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

// ToPgInt8 converts int64 to pgtype.Int8.
func ToPgInt8(i int64) pgtype.Int8 {
	return pgtype.Int8{Int64: i, Valid: true}
}

// ToPgInt8Ptr converts *int64 to pgtype.Int8.
func ToPgInt8Ptr(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

// FromPgInt8 converts pgtype.Int8 to int64.
func FromPgInt8(v pgtype.Int8) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}

// FromPgInt8Ptr converts pgtype.Int8 to *int64 (nullable).
func FromPgInt8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

// ToPgTimestamptzPtr converts a *time.Time to pgtype.Timestamptz.
func ToPgTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// FromPgTimestamptz converts pgtype.Timestamptz to *time.Time.
func FromPgTimestamptz(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}

// ToPgNumeric converts float64 to pgtype.Numeric.
// 全精度序列化（-1 = 最短精确表示）——转换器不做业务舍入；需要两位小数
// 的列约束由 DB numeric(x,2) 或业务层负责（曾用 %.2f 静默截断，汇率/
// 成本类两位以上小数在入库前就被抹掉）。
func ToPgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(f, 'f', -1, 64))
	return n
}

// ToPgNumericPtr converts *float64 to pgtype.Numeric.
func ToPgNumericPtr(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	return ToPgNumeric(*f)
}

// FromPgNumeric converts pgtype.Numeric to float64.
func FromPgNumeric(v pgtype.Numeric) float64 {
	if !v.Valid {
		return 0
	}
	f64, _ := v.Float64Value()
	return f64.Float64
}

// FromPgUUID converts pgtype.UUID to string.
func FromPgUUID(v pgtype.UUID) string {
	if !v.Valid {
		return ""
	}
	return uuid.UUID(v.Bytes).String()
}

// ToPgUUID converts a *string (UUID) to pgtype.UUID.
func ToPgUUID(s *string) pgtype.UUID {
	if s == nil || *s == "" {
		return pgtype.UUID{Valid: false}
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Valid: true, Bytes: [16]byte(id)}
}

// ToPgUUIDFromUUID converts a uuid.UUID to pgtype.UUID.
func ToPgUUIDFromUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Valid: true, Bytes: id}
}

// ToPgUUIDPtr converts *uuid.UUID to pgtype.UUID.
func ToPgUUIDPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

// FromPgUUIDToPtr converts pgtype.UUID to *uuid.UUID.
func FromPgUUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}
