package cmd

import (
	"github.com/mnh-shop/7alosclaw/internal/agent"
	"github.com/mnh-shop/7alosclaw/internal/audio"
	"github.com/mnh-shop/7alosclaw/internal/bus"
	"github.com/mnh-shop/7alosclaw/internal/cache"
	"github.com/mnh-shop/7alosclaw/internal/channelmemory"
	"github.com/mnh-shop/7alosclaw/internal/channels"
	"github.com/mnh-shop/7alosclaw/internal/config"
	"github.com/mnh-shop/7alosclaw/internal/eventbus"
	"github.com/mnh-shop/7alosclaw/internal/gateway"
	httpapi "github.com/mnh-shop/7alosclaw/internal/http"
	"github.com/mnh-shop/7alosclaw/internal/memory"
	"github.com/mnh-shop/7alosclaw/internal/providers"
	"github.com/mnh-shop/7alosclaw/internal/skills"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/internal/tools"
	usagecaps "github.com/mnh-shop/7alosclaw/internal/usage/caps"
	"github.com/mnh-shop/7alosclaw/internal/vault"
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
