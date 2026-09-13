package pg

import (
	"errors"
	"fmt"

	"github.com/mnh-shop/7alosclaw/internal/config"
	"github.com/mnh-shop/7alosclaw/internal/crypto"
	"github.com/mnh-shop/7alosclaw/internal/store"
)

// ErrEncryptionKeyRequired is returned when the PostgreSQL backend is started
// without a usable encryption key. The multi-tenant/PG path fails closed:
// provider API keys and other credentials must never be stored in plaintext.
var ErrEncryptionKeyRequired = errors.New("postgres backend: ZALOSCLAW_ENCRYPTION_KEY is required")

// NewPGStores creates all stores backed by Postgres.
func NewPGStores(cfg store.StoreConfig) (*store.Stores, error) {
	// Fail closed: refusing to start is safer than silently persisting secrets
	// (provider API keys, tokens) as plaintext in the shared multi-tenant DB.
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("%w: refusing to start with plaintext secret storage; set it to a 32-byte key (64 hex chars, 44-char base64, or raw 32 bytes)", ErrEncryptionKeyRequired)
	}
	if _, err := crypto.DeriveKey(cfg.EncryptionKey); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionKeyRequired, err)
	}

	db, err := OpenDB(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	initSqlx(db)

	memCfg := DefaultPGMemoryConfig()

	skillsDir := cfg.SkillsStorageDir
	if skillsDir == "" {
		skillsDir = config.ResolvedDataDirFromEnv() + "/skills-store"
	}
	subagentTasks := NewPGSubagentTaskStore(db)

	pgStores := &store.Stores{
		DB:                     db,
		Sessions:               NewPGSessionStore(db),
		Memory:                 NewPGMemoryStore(db, memCfg),
		Cron:                   NewPGCronStore(db),
		Pairing:                NewPGPairingStore(db),
		Skills:                 NewPGSkillStore(db, skillsDir),
		Agents:                 NewPGAgentStore(db),
		Providers:              NewPGProviderStore(db, cfg.EncryptionKey),
		Tracing:                NewPGTracingStore(db),
		RunTimeline:            NewPGRunTimelineStore(db),
		MCP:                    NewPGMCPServerStore(db, cfg.EncryptionKey),
		MCPOAuthTokens:         NewPGMCPOAuthTokenStore(db, cfg.EncryptionKey),
		ChannelInstances:       NewPGChannelInstanceStore(db, cfg.EncryptionKey),
		ConfigSecrets:          NewPGConfigSecretsStore(db, cfg.EncryptionKey),
		AgentLinks:             NewPGAgentLinkStore(db),
		Teams:                  NewPGTeamStore(db),
		BuiltinTools:           NewPGBuiltinToolStore(db),
		PendingMessages:        NewPGPendingMessageStore(db),
		ChannelMemory:          NewPGChannelMemoryExtractionStore(db),
		KnowledgeGraph:         NewPGKnowledgeGraphStore(db),
		Contacts:               NewPGContactStore(db),
		Activity:               NewPGActivityStore(db),
		Snapshots:              NewPGSnapshotStore(db),
		UsageEvents:            NewPGUsageEventStore(db),
		BrowserCookies:         NewPGBrowserCookieStore(db, cfg.EncryptionKey),
		SecureCLI:              NewPGSecureCLIStore(db, cfg.EncryptionKey),
		SecureCLIGrants:        NewPGSecureCLIAgentGrantStore(db, cfg.EncryptionKey),
		APIKeys:                NewPGAPIKeyStore(db),
		Heartbeats:             NewPGHeartbeatStore(db),
		ConfigPermissions:      NewPGConfigPermissionStore(db),
		Tenants:                NewPGTenantStore(db),
		BuiltinToolTenantCfgs:  NewPGBuiltinToolTenantConfigStore(db),
		SkillTenantCfgs:        NewPGSkillTenantConfigStore(db),
		SkillEvolution:         NewPGSkillEvolutionStore(db),
		SystemConfigs:          NewPGSystemConfigStore(db),
		SubagentTasks:          subagentTasks,
		SubagentTaskRecovery:   subagentTasks,
		Vault:                  NewPGVaultStore(db),
		Episodic:               NewPGEpisodicStore(db),
		EvolutionMetrics:       NewPGEvolutionMetricsStore(db),
		EvolutionSuggestions:   NewPGEvolutionSuggestionStore(db),
		BitrixPortals:          NewPGBitrixPortalStore(db, cfg.EncryptionKey),
		Hooks:                  NewPGHookStore(db),
		Webhooks:               NewPGWebhookStore(db),
		WebhookCalls:           NewPGWebhookCallStore(db),
		Workstations:           NewPGWorkstationStore(db, cfg.EncryptionKey),
		WorkstationLinks:       NewPGAgentWorkstationLinkStore(db),
		WorkstationPermissions: NewPGWorkstationPermissionStore(db),
		WorkstationActivity:    NewPGWorkstationActivityStore(db),
		UsageCaps:              NewPGUsageCapStore(db),
	}
	// Wire permStore into WorkstationStore so Create seeds allowlist atomically (H5 fix).
	// Must happen after both stores are constructed.
	pgStores.Workstations.(*PGWorkstationStore).SetPermStore(pgStores.WorkstationPermissions)
	return pgStores, nil
}
