//go:build !sqlite && !sqliteonly

package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mnh-shop/7alosclaw/internal/bus"
	"github.com/mnh-shop/7alosclaw/internal/config"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/internal/store/pg"
	"github.com/mnh-shop/7alosclaw/internal/tracing"
)

// setupStoresAndTracing creates PG stores, tracing collector, snapshot worker, and wires cron config.
// This is the default (PG-only) build — no SQLite support compiled in.
func setupStoresAndTracing(
	cfg *config.Config,
	dataDir string,
	msgBus *bus.MessageBus,
) (*store.Stores, *tracing.Collector, *tracing.SnapshotWorker) {
	if cfg.Database.PostgresDSN == "" {
		slog.Error("ZALOSCLAW_POSTGRES_DSN is required. Set it in your environment or .env.local file.")
		os.Exit(1)
	}

	if err := checkSchemaOrAutoUpgrade(cfg.Database.PostgresDSN); err != nil {
		slog.Error("schema compatibility check failed", "error", err)
		os.Exit(1)
	}

	storeCfg := store.StoreConfig{
		PostgresDSN:      cfg.Database.PostgresDSN,
		EncryptionKey:    os.Getenv("ZALOSCLAW_ENCRYPTION_KEY"),
		SkillsStorageDir: filepath.Join(dataDir, "skills-store"),
	}
	pgStores, pgErr := pg.NewPGStores(storeCfg)
	if pgErr != nil {
		slog.Error("failed to create PG stores", "error", pgErr)
		os.Exit(1)
	}

	traceCollector, snapshotWorker := wireTracingAndCron(cfg, pgStores, msgBus, dataDir)
	return pgStores, traceCollector, snapshotWorker
}
