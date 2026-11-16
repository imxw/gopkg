package pgxutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	"github.com/imxw/gopkg/errorx"
)

func pgErr(code, constraint, table string) *pgconn.PgError {
	return &pgconn.PgError{Code: code, ConstraintName: constraint, TableName: table}
}

var errNotFound = errorx.New(5601, 404, "APP_NOT_FOUND", "app not found")
var errConflict = errorx.New(5602, 409, "APP_NAME_CONFLICT", "name conflict")
var errFKBlocked = errorx.New(5603, 409, "APP_HAS_INSTANCES", "app has instances")

func TestMapPgxError_Nil(t *testing.T) {
	assert.Nil(t, MapPgxError(nil, errNotFound))
}

func TestMapPgxError_NoRows(t *testing.T) {
	wrapped := fmt.Errorf("query by id: %w", pgx.ErrNoRows)
	got := MapPgxError(wrapped, errNotFound)
	assert.Equal(t, errNotFound, got)
	assert.ErrorIs(t, got, errNotFound)
}

func TestMapPgxError_NoRowsNilFallback(t *testing.T) {
	got := MapPgxError(pgx.ErrNoRows, nil)
	assert.Equal(t, errorx.ErrNotFound, got)
}

func TestMapPgxError_UniqueViolationAnyRule(t *testing.T) {
	got := MapPgxError(pgErr(errCodeUniqueViolation, "apps_name_key", "apps"), errNotFound,
		WithUniqueViolation(errConflict))
	assert.Equal(t, errConflict, got)
}

func TestMapPgxError_UniqueViolationConstraintSpecificWins(t *testing.T) {
	otherConflict := errorx.New(5604, 409, "APP_CODE_CONFLICT", "code conflict")
	got := MapPgxError(pgErr(errCodeUniqueViolation, "apps_code_key", "apps"), errNotFound,
		WithUniqueViolation(errConflict),
		WithUniqueViolationOn("apps_code_key", otherConflict))
	assert.Equal(t, otherConflict, got)
}

func TestMapPgxError_UniqueViolationMissedConstraintFallsBack(t *testing.T) {
	got := MapPgxError(pgErr(errCodeUniqueViolation, "apps_owner_idx", "apps"), errNotFound,
		WithUniqueViolation(errConflict),
		WithUniqueViolationOn("apps_code_key", errorx.New(5604, 409, "X", "x")))
	assert.Equal(t, errConflict, got)
}

func TestMapPgxError_FKViolationTableRule(t *testing.T) {
	got := MapPgxError(pgErr(errCodeFKViolation, "instances_app_fk", "instances"), errNotFound,
		WithFKViolationOn("instances", errFKBlocked))
	assert.Equal(t, errFKBlocked, got)
}

func TestMapPgxError_FKViolationAnyRule(t *testing.T) {
	got := MapPgxError(pgErr(errCodeFKViolation, "instances_app_fk", "instances"), errNotFound,
		WithFKViolation(errFKBlocked))
	assert.Equal(t, errFKBlocked, got)
}

func TestMapPgxError_UnmatchedErrorPassesThrough(t *testing.T) {
	boom := errors.New("connection refused")
	got := MapPgxError(boom, errNotFound, WithUniqueViolation(errConflict))
	assert.Equal(t, boom, got)

	// 非命中约束名的 unique 违反且无 any 规则 → 原样透传。
	got2 := MapPgxError(pgErr(errCodeUniqueViolation, "x_key", "t"), errNotFound,
		WithUniqueViolationOn("other_key", errConflict))
	assert.NotEqual(t, errConflict, got2)
}

func TestMapPgxError_WrappedPgError(t *testing.T) {
	wrapped := fmt.Errorf("insert app: %w", pgErr(errCodeUniqueViolation, "apps_name_key", "apps"))
	got := MapPgxError(wrapped, errNotFound, WithUniqueViolation(errConflict))
	assert.Equal(t, errConflict, got)
}

func TestIsUniqueViolationOn_IsFKViolationOn(t *testing.T) {
	uq := pgErr(errCodeUniqueViolation, "apps_name_key", "apps")
	assert.True(t, IsUniqueViolationOn(uq, "apps_name_key"))
	assert.False(t, IsUniqueViolationOn(uq, "other_key"))
	assert.False(t, IsUniqueViolationOn(uq, ""))

	fk := pgErr(errCodeFKViolation, "instances_app_fk", "instances")
	assert.True(t, IsFKViolationOn(fk, "instances"))
	assert.False(t, IsFKViolationOn(fk, "hosts"))

	assert.False(t, IsUniqueViolationOn(errors.New("boom"), "apps_name_key"))
	assert.False(t, IsFKViolationOn(errors.New("boom"), "instances"))
}
