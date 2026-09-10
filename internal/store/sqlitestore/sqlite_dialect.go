//go:build sqlite || sqliteonly

package sqlitestore

import "github.com/mnh-shop/7alosclaw/internal/store/base"

// sqliteDialect implements base.Dialect for SQLite (? placeholders + value transform).
var sqliteDialect base.Dialect = sqliteDialectImpl{}

type sqliteDialectImpl struct{}

func (sqliteDialectImpl) Placeholder(_ int) string { return "?" }
func (sqliteDialectImpl) TransformValue(v any) any { return sqliteVal(v) }
func (sqliteDialectImpl) SupportsReturning() bool  { return false }
