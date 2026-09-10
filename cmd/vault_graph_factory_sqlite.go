//go:build sqlite || sqliteonly

package cmd

import (
	"database/sql"

	"github.com/nextlevelbuilder/zalosclaw/internal/store"
	"github.com/nextlevelbuilder/zalosclaw/internal/store/sqlitestore"
)

func newVaultGraphStore(db *sql.DB) store.VaultGraphStore { return sqlitestore.NewSQLiteVaultGraphStore(db) }
func newKGGraphStore(db *sql.DB) store.KGGraphStore       { return sqlitestore.NewSQLiteKGGraphStore(db) }
