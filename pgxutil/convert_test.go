package pgxutil

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 全量转换面表格测试：往返语义 + 零值/nil 的 Valid 边界。
// pgtype 的 Valid 边界是本包存在的唯一理由，故每族都钉「空值进 →
// Invalid 出 → 回程零值」三段。

func TestTextRoundTrip(t *testing.T) {
	assert.Equal(t, pgtype.Text{String: "x", Valid: true}, ToPgText("x"))
	assert.Equal(t, pgtype.Text{}, ToPgText(""), "空串 = Invalid（列 NULL 语义）")
	s := "x"
	assert.Equal(t, pgtype.Text{String: "x", Valid: true}, ToPgTextPtr(&s))
	assert.Equal(t, pgtype.Text{}, ToPgTextPtr(nil))
	assert.Equal(t, "x", FromPgText(pgtype.Text{String: "x", Valid: true}))
	assert.Equal(t, "", FromPgText(pgtype.Text{}))
}

func TestBoolRoundTrip(t *testing.T) {
	assert.Equal(t, pgtype.Bool{Bool: true, Valid: true}, ToPgBool(true))
	assert.Equal(t, true, FromPgBool(ToPgBool(true)))
	assert.Equal(t, false, FromPgBool(pgtype.Bool{}), "Invalid 回程 false")
	b := true
	assert.Equal(t, pgtype.Bool{Bool: true, Valid: true}, ToPgBoolPtr(&b))
	assert.Equal(t, pgtype.Bool{}, ToPgBoolPtr(nil))
}

func TestIntRoundTrip(t *testing.T) {
	assert.Equal(t, pgtype.Int4{Int32: 7, Valid: true}, ToPgInt4(7))
	assert.Equal(t, int32(7), FromPgInt4(ToPgInt4(7)))
	assert.Equal(t, int32(0), FromPgInt4(pgtype.Int4{}))
	i := int32(7)
	assert.Equal(t, pgtype.Int4{Int32: 7, Valid: true}, ToPgInt4Ptr(&i))
	assert.Equal(t, pgtype.Int4{}, ToPgInt4Ptr(nil))

	assert.Equal(t, pgtype.Int8{Int64: 9, Valid: true}, ToPgInt8(9))
	assert.Equal(t, int64(9), FromPgInt8(ToPgInt8(9)))
	assert.Equal(t, int64(0), FromPgInt8(pgtype.Int8{}))
	i8 := int64(9)
	assert.Equal(t, pgtype.Int8{Int64: 9, Valid: true}, ToPgInt8Ptr(&i8))
	assert.Equal(t, pgtype.Int8{}, ToPgInt8Ptr(nil))
	got := FromPgInt8Ptr(pgtype.Int8{Int64: 9, Valid: true})
	require.NotNil(t, got)
	assert.Equal(t, int64(9), *got)
	assert.Nil(t, FromPgInt8Ptr(pgtype.Int8{}))
}

func TestTimestamptzRoundTrip(t *testing.T) {
	now := time.Now()
	assert.Equal(t, pgtype.Timestamptz{Time: now, Valid: true}, ToPgTimestamptzPtr(&now))
	assert.Equal(t, pgtype.Timestamptz{}, ToPgTimestamptzPtr(nil))
	got := FromPgTimestamptz(pgtype.Timestamptz{Time: now, Valid: true})
	require.NotNil(t, got)
	assert.True(t, got.Equal(now))
	assert.Nil(t, FromPgTimestamptz(pgtype.Timestamptz{}))
}

// Numeric 全精度往返：ToPgNumeric 不得静默截断小数位（曾用 %.2f 格式化，
// 汇率/成本类两位以上小数在入库前就被抹掉——转换器不做业务舍入）。
func TestNumericRoundTrip_FullPrecision(t *testing.T) {
	for _, f := range []float64{3.14159, 0.005, 1234.5678} {
		n := ToPgNumeric(f)
		got := FromPgNumeric(n)
		assert.InDelta(t, f, got, 1e-9, "%v 不得被截断到两位小数", f)
	}
	assert.Equal(t, pgtype.Numeric{}, ToPgNumericPtr(nil))
	assert.Equal(t, float64(0), FromPgNumeric(pgtype.Numeric{}))
}

func TestUUIDRoundTrip(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	assert.Equal(t, id.String(), FromPgUUID(pgtype.UUID{Bytes: id, Valid: true}))
	assert.Equal(t, "", FromPgUUID(pgtype.UUID{}))

	s := id.String()
	assert.Equal(t, pgtype.UUID{Bytes: id, Valid: true}, ToPgUUID(&s))
	invalid := "not-a-uuid"
	assert.Equal(t, pgtype.UUID{}, ToPgUUID(&invalid), "非法 UUID fail-closed Invalid（不 panic）")
	empty := ""
	assert.Equal(t, pgtype.UUID{}, ToPgUUID(&empty))
	assert.Equal(t, pgtype.UUID{}, ToPgUUID(nil))

	assert.Equal(t, pgtype.UUID{Bytes: id, Valid: true}, ToPgUUIDFromUUID(id))
	u := id
	assert.Equal(t, pgtype.UUID{Bytes: id, Valid: true}, ToPgUUIDPtr(&u))
	assert.Equal(t, pgtype.UUID{}, ToPgUUIDPtr(nil))
	back := FromPgUUIDToPtr(pgtype.UUID{Bytes: id, Valid: true})
	require.NotNil(t, back)
	assert.Equal(t, id, *back)
	assert.Nil(t, FromPgUUIDToPtr(pgtype.UUID{}))
}
