package cmd

import (
	"sync"

	"github.com/nextlevelbuilder/zalosclaw/internal/agent"
	"github.com/nextlevelbuilder/zalosclaw/internal/bus"
	"github.com/nextlevelbuilder/zalosclaw/internal/channels"
	"github.com/nextlevelbuilder/zalosclaw/internal/config"
	"github.com/nextlevelbuilder/zalosclaw/internal/memory"
	"github.com/nextlevelbuilder/zalosclaw/internal/providers"
	"github.com/nextlevelbuilder/zalosclaw/internal/scheduler"
	"github.com/nextlevelbuilder/zalosclaw/internal/store"
	"github.com/nextlevelbuilder/zalosclaw/internal/tools"
	usagecaps "github.com/nextlevelbuilder/zalosclaw/internal/usage/caps"
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
