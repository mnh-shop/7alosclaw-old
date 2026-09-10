package cmd

import (
	"github.com/nextlevelbuilder/zalosclaw/internal/agent"
	"github.com/nextlevelbuilder/zalosclaw/internal/audio"
	"github.com/nextlevelbuilder/zalosclaw/internal/bus"
	"github.com/nextlevelbuilder/zalosclaw/internal/cache"
	"github.com/nextlevelbuilder/zalosclaw/internal/channelmemory"
	"github.com/nextlevelbuilder/zalosclaw/internal/channels"
	"github.com/nextlevelbuilder/zalosclaw/internal/config"
	"github.com/nextlevelbuilder/zalosclaw/internal/eventbus"
	"github.com/nextlevelbuilder/zalosclaw/internal/gateway"
	httpapi "github.com/nextlevelbuilder/zalosclaw/internal/http"
	"github.com/nextlevelbuilder/zalosclaw/internal/memory"
	"github.com/nextlevelbuilder/zalosclaw/internal/providers"
	"github.com/nextlevelbuilder/zalosclaw/internal/skills"
	"github.com/nextlevelbuilder/zalosclaw/internal/store"
	"github.com/nextlevelbuilder/zalosclaw/internal/tools"
	usagecaps "github.com/nextlevelbuilder/zalosclaw/internal/usage/caps"
	"github.com/nextlevelbuilder/zalosclaw/internal/vault"
)

// gatewayDeps holds shared dependencies used across the extracted gateway setup functions.
// It is populated in runGateway() and passed to helper methods to avoid long parameter lists.
type gatewayDeps struct {
	cfg              *config.Config
	server           *gateway.Server
	msgBus           *bus.MessageBus
	pgStores         *store.Stores
	providerRegistry *providers.Registry
	channelMgr       *channels.Manager
	channelMemorySvc *channelmemory.Service
	agentRouter      *agent.Router
	toolsReg         *tools.Registry
	skillsLoader     *skills.Loader         // optional: enables skill creation in evolution approval
	permCache        *cache.PermissionCache // nil if no tenant store; closed on shutdown to stop sweep goroutines
	enrichProgress   *vault.EnrichProgress  // nil if enrichment worker not registered
	enrichWorker     *vault.EnrichWorker    // nil if enrichment worker not registered; for stop/enqueue
	workspace        string
	dataDir          string
	domainBus        eventbus.DomainEventBus
	usageCapSvc      *usagecaps.Service
	audioMgr         *audio.Manager      // nil if TTS not configured; used by TTSHandler
	ttsHandler       *httpapi.TTSHandler // nil if TTS not configured; for hot-reload
	teamWorkEmbedder memory.EmbeddingProvider
}
