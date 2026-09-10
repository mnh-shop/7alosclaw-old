//go:build sqlite || sqliteonly

package cmd

import (
	"database/sql"

	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/internal/store/sqlitestore"
)

func newVaultGraphStore(db *sql.DB) store.VaultGraphStore { return sqlitestore.NewSQLiteVaultGraphStore(db) }
func newKGGraphStore(db *sql.DB) store.KGGraphStore       { return sqlitestore.NewSQLiteKGGraphStore(db) }
