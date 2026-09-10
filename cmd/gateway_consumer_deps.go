package cmd

import (
	"sync"

	"github.com/mnh-shop/7alosclaw/internal/agent"
	"github.com/mnh-shop/7alosclaw/internal/bus"
	"github.com/mnh-shop/7alosclaw/internal/channels"
	"github.com/mnh-shop/7alosclaw/internal/config"
	"github.com/mnh-shop/7alosclaw/internal/memory"
	"github.com/mnh-shop/7alosclaw/internal/providers"
	"github.com/mnh-shop/7alosclaw/internal/scheduler"
	"github.com/mnh-shop/7alosclaw/internal/store"
	"github.com/mnh-shop/7alosclaw/internal/tools"
	usagecaps "github.com/mnh-shop/7alosclaw/internal/usage/caps"
)

// ConsumerDeps bundles shared dependencies for consumer message handlers.
// Replaces 11+ positional params with a single injectable struct.
type ConsumerDeps struct {
	Cfg              *config.Config
	Agents           *agent.Router
	Sched            *scheduler.Scheduler
	ChannelMgr       *channels.Manager
	MsgBus           *bus.MessageBus
	TeamStore        store.TeamStore
	AgentLinkStore   store.AgentLinkStore
	AgentStore       store.AgentStore
	SessStore        store.SessionStore
	PostTurn         tools.PostTurnProcessor
	QuotaChecker     *channels.QuotaChecker
	ContactCollector *store.ContactCollector
	TaskRunSessions  sync.Map
	SubagentMgr      *tools.SubagentManager
	UsageCaps        *usagecaps.Service
	ProviderReg      *providers.Registry
	TeamWorkEmbedder memory.EmbeddingProvider
	BgWg             sync.WaitGroup
	GetAnnounceMu    func(string) *sync.Mutex
}
