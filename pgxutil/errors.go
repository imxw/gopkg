package pgxutil

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL constraint-violation SQLSTATE codes.
const (
	errCodeUniqueViolation = "23505"
	errCodeFKViolation     = "23503"
)

// IsUniqueViolation checks if the error is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == errCodeUniqueViolation
}

// IsUniqueViolationOn checks a unique violation raised by the named constraint.
func IsUniqueViolationOn(err error, constraint string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == errCodeUniqueViolation && pgErr.ConstraintName == constraint
}

// IsFKViolation checks if the error is a PostgreSQL foreign key constraint violation (SQLSTATE 23503).
func IsFKViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == errCodeFKViolation
}

// IsFKViolationOn checks an FK violation referencing the named table.
func IsFKViolationOn(err error, table string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == errCodeFKViolation && pgErr.TableName == table
}
