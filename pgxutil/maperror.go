package pgxutil

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/imxw/gopkg/errorx"
)

// MapPgxError 把 pgx 驱动错误翻译为业务 error（repo 层统一出口）：
//
//   - pgx.ErrNoRows（含包装链）→ notFound 原样返回；传 nil 时兜底
//     errorx.ErrNotFound。
//   - 唯一约束/FK 违反（SQLSTATE 23505/23503）→ 按 MapOption 命中规则
//     返回对应冲突错误；约束名/表名感知规则优先于 any 规则。
//   - 其余错误原样返回（调用方的 fmt.Errorf 上下文包装保持完整）。
//
// 典型用法（一行替换三行 if 块）：
//
//	return pgxutil.MapPgxError(err, fmt.Errorf("get app %d: %w", id, app.ErrAppNotFound))
func MapPgxError(err error, notFound error, opts ...MapOption) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		if notFound == nil {
			return errorx.ErrNotFound
		}
		return notFound
	}

	cfg := &mapConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}

	switch pgErr.Code {
	case errCodeUniqueViolation:
		// 约束名感知规则优先（更精确的错误语义），any 规则兜底。
		if mapped, hit := cfg.uniqueOn[pgErr.ConstraintName]; hit {
			return mapped
		}
		if cfg.uniqueAny != nil {
			return cfg.uniqueAny
		}
	case errCodeFKViolation:
		if mapped, hit := cfg.fkOn[pgErr.TableName]; hit {
			return mapped
		}
		if cfg.fkAny != nil {
			return cfg.fkAny
		}
	}
	return err
}

// MapOption 定制 MapPgxError 对约束违反的翻译规则。
type MapOption func(*mapConfig)

type mapConfig struct {
	uniqueAny error
	uniqueOn  map[string]error
	fkAny     error
	fkOn      map[string]error
}

// WithUniqueViolation 任意唯一约束违反（23505）→ err。
func WithUniqueViolation(err error) MapOption {
	return func(c *mapConfig) { c.uniqueAny = err }
}

// WithUniqueViolationOn 指定约束名的唯一违反 → err（优先于 any 规则）。
func WithUniqueViolationOn(constraint string, err error) MapOption {
	return func(c *mapConfig) {
		if c.uniqueOn == nil {
			c.uniqueOn = make(map[string]error)
		}
		c.uniqueOn[constraint] = err
	}
}

// WithFKViolation 任意外键违反（23503）→ err。
func WithFKViolation(err error) MapOption {
	return func(c *mapConfig) { c.fkAny = err }
}

// WithFKViolationOn 指定表的外键违反 → err（优先于 any 规则）。
func WithFKViolationOn(table string, err error) MapOption {
	return func(c *mapConfig) {
		if c.fkOn == nil {
			c.fkOn = make(map[string]error)
		}
		c.fkOn[table] = err
	}
}
