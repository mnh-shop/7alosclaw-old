import { Suspense } from "react";
import { Routes, Route, Navigate } from "react-router";
import { AppLayout } from "@/components/layout/app-layout";
import { RequireAuth } from "@/components/shared/require-auth";
import { RequireAdmin, RequireOwner } from "@/components/shared/require-role";
import { RequireSetup } from "@/components/shared/require-setup";
import { ErrorBoundary } from "@/components/shared/error-boundary";
import { ROUTES } from "@/lib/constants";
import { lazyWithRetry } from "@/lib/lazy-with-retry";

// Lazy-loaded pages
const LoginPage = lazyWithRetry(() =>
  import("@/pages/login/login-page").then((m) => ({ default: m.LoginPage })),
);
const OverviewPage = lazyWithRetry(() =>
  import("@/pages/overview/overview-page").then((m) => ({ default: m.OverviewPage })),
);
const ChatPage = lazyWithRetry(() =>
  import("@/pages/chat/chat-page").then((m) => ({ default: m.ChatPage })),
);
const AgentsPage = lazyWithRetry(() =>
  import("@/pages/agents/agents-page").then((m) => ({ default: m.AgentsPage })),
);
const AgentCodexPoolPage = lazyWithRetry(() =>
  import("@/pages/agents/agent-detail/agent-codex-pool-page").then((m) => ({ default: m.AgentCodexPoolPage })),
);
const ImportExportPage = lazyWithRetry(() =>
  import("@/pages/import-export/import-export-page").then((m) => ({ default: m.ImportExportPage })),
);
const SessionsPage = lazyWithRetry(() =>
  import("@/pages/sessions/sessions-page").then((m) => ({ default: m.SessionsPage })),
);
const SkillsPage = lazyWithRetry(() =>
  import("@/pages/skills/skills-page").then((m) => ({ default: m.SkillsPage })),
);
const CronPage = lazyWithRetry(() =>
  import("@/pages/cron/cron-page").then((m) => ({ default: m.CronPage })),
);
const TracesPage = lazyWithRetry(() =>
  import("@/pages/traces/traces-page").then((m) => ({ default: m.TracesPage })),
);
const UsagePage = lazyWithRetry(() =>
  import("@/pages/usage/usage-page").then((m) => ({ default: m.UsagePage })),
);
const WorkstationsPage = lazyWithRetry(() =>
  import("@/pages/workstations/workstations-page").then((m) => ({ default: m.WorkstationsPage })),
);
const ChannelsPage = lazyWithRetry(() =>
  import("@/pages/channels/channels-page").then((m) => ({ default: m.ChannelsPage })),
);
const ApprovalsPage = lazyWithRetry(() =>
  import("@/pages/approvals/approvals-page").then((m) => ({ default: m.ApprovalsPage })),
);
const MCPPage = lazyWithRetry(() =>
  import("@/pages/mcp/mcp-page").then((m) => ({ default: m.MCPPage })),
);
const TeamsPage = lazyWithRetry(() =>
  import("@/pages/teams/teams-page").then((m) => ({ default: m.TeamsPage })),
);
const StoragePage = lazyWithRetry(() =>
  import("@/pages/storage/storage-page").then((m) => ({ default: m.StoragePage })),
);
const SetupPage = lazyWithRetry(() =>
  import("@/pages/setup/setup-page").then((m) => ({ default: m.SetupPage })),
);
const PendingMessagesPage = lazyWithRetry(() =>
  import("@/pages/pending-messages/pending-messages-page").then((m) => ({ default: m.PendingMessagesPage })),
);
const MemoryPage = lazyWithRetry(() =>
  import("@/pages/memory/memory-page").then((m) => ({ default: m.MemoryPage })),
);
const VaultPage = lazyWithRetry(() =>
  import("@/pages/vault/vault-page").then((m) => ({ default: m.VaultPage })),
);
const KnowledgeGraphPage = lazyWithRetry(() =>
  import("@/pages/knowledge-graph/knowledge-graph-page").then((m) => ({ default: m.KnowledgeGraphPage })),
);
const ContactsPage = lazyWithRetry(() =>
  import("@/pages/contacts/contacts-page").then((m) => ({ default: m.ContactsPage })),
);
const OptionsPage = lazyWithRetry(() =>
  import("@/pages/options/options-page").then((m) => ({ default: m.OptionsPage })),
);
const TenantSelectorPage = lazyWithRetry(() =>
  import("@/pages/login/tenant-selector").then((m) => ({ default: m.TenantSelectorPage })),
);

// Operator-only surfaces removed from the customer/organisation dashboard.
// The page files are kept; these paths just bounce back to Overview so a typed
// URL cannot reach them. This is the single place that defines the hidden set.
const HIDDEN_ROUTES = [
  ROUTES.PROVIDERS,
  ROUTES.PROVIDER_DETAIL,
  ROUTES.API_KEYS,
  // Was a bespoke redirect into `/packages?tab=cli-credentials`, but /packages is
  // itself hidden — two hops to nowhere. Hidden like the rest, so a typed URL
  // lands on Overview in one step. The page files are kept.
  ROUTES.CLI_CREDENTIALS,
  ROUTES.NODES,
  ROUTES.TTS,
  // Inbound-only feature: with no route from off-node to a guest, an external
  // service can never call a webhook. Dropped, not merely hidden.
  ROUTES.WEBHOOKS,
  ROUTES.TENANTS,
  ROUTES.TENANT_DETAIL,
  ROUTES.CONFIG,
  ROUTES.BUILTIN_TOOLS,
  ROUTES.HOOKS,
  ROUTES.HOOK_DETAIL,
  ROUTES.ACTIVITY,
  ROUTES.EVENTS,
  ROUTES.LOGS,
  ROUTES.PACKAGES,
  ROUTES.BACKUP_RESTORE,
  // WORKSTATION_DETAIL is vestigial: there has never been a detail route or page
  // (the list handles everything in dialogs), so it stays hidden and a typed URL
  // lands on Overview. ROUTES.WORKSTATIONS itself is live — see the route table.
  ROUTES.WORKSTATION_DETAIL,
] as const;

function PageLoader() {
  return (
    <div className="flex h-full items-center justify-center">
      <img src="/7alos-mark.svg" alt="" className="h-8 w-8 animate-pulse opacity-50" />
    </div>
  );
}

export function AppRoutes() {
  return (
    <ErrorBoundary>
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route path={ROUTES.LOGIN} element={<LoginPage />} />

        {/* Tenant selector — accessible when authenticated but tenant not yet selected */}
        <Route path={ROUTES.SELECT_TENANT} element={<TenantSelectorPage />} />

        {/* Setup wizard — standalone layout, requires auth but no sidebar */}
        <Route
          path={ROUTES.SETUP}
          element={
            <RequireAuth>
              <SetupPage />
            </RequireAuth>
          }
        />

        {/* Main app — requires auth + setup complete */}
        <Route
          element={
            <RequireAuth>
              <RequireSetup>
                <AppLayout />
              </RequireSetup>
            </RequireAuth>
          }
        >
          <Route index element={<Navigate to={ROUTES.OVERVIEW} replace />} />
          <Route path={ROUTES.OVERVIEW} element={<OverviewPage />} />
          <Route path={ROUTES.CHAT_PATTERN} element={<ChatPage />} />
          <Route path={ROUTES.AGENTS} element={<AgentsPage key="list" />} />
          <Route path={ROUTES.IMPORT_EXPORT} element={<RequireOwner><ImportExportPage /></RequireOwner>} />
          <Route path={ROUTES.AGENT_CODEX_POOL} element={<RequireAdmin><AgentCodexPoolPage /></RequireAdmin>} />
          <Route path={ROUTES.AGENT_DETAIL} element={<AgentsPage key="detail" />} />
          <Route path={ROUTES.TEAMS} element={<TeamsPage key="list" />} />
          <Route path={ROUTES.TEAM_DETAIL} element={<TeamsPage key="detail" />} />
          <Route path={ROUTES.SESSIONS} element={<SessionsPage key="list" />} />
          <Route path={ROUTES.SESSION_DETAIL} element={<SessionsPage key="detail" />} />
          <Route path={ROUTES.SKILLS} element={<SkillsPage key="list" />} />
          <Route path={ROUTES.SKILL_DETAIL} element={<SkillsPage key="detail" />} />
          <Route path={ROUTES.CRON} element={<CronPage key="list" />} />
          <Route path={ROUTES.CRON_DETAIL} element={<CronPage key="detail" />} />
          <Route path={ROUTES.CHANNELS} element={<RequireAdmin><ChannelsPage key="list" /></RequireAdmin>} />
          <Route path={ROUTES.CHANNEL_DETAIL} element={<RequireAdmin><ChannelsPage key="detail" /></RequireAdmin>} />
          <Route path={ROUTES.MCP} element={<RequireAdmin><MCPPage /></RequireAdmin>} />
          <Route path={ROUTES.STORAGE} element={<RequireAdmin><StoragePage /></RequireAdmin>} />
          <Route path={ROUTES.WORKSTATIONS} element={<RequireAdmin><WorkstationsPage /></RequireAdmin>} />
          <Route path={ROUTES.TRACES} element={<TracesPage key="list" />} />
          <Route path={ROUTES.TRACE_DETAIL} element={<TracesPage key="detail" />} />
          <Route path={ROUTES.USAGE} element={<UsagePage />} />
          <Route path={ROUTES.CONTACTS} element={<ContactsPage />} />
          <Route path={ROUTES.APPROVALS} element={<ApprovalsPage />} />
          <Route path={ROUTES.OPTIONS} element={<OptionsPage />} />
          <Route path={ROUTES.PENDING_MESSAGES} element={<PendingMessagesPage />} />
          <Route path={ROUTES.MEMORY} element={<MemoryPage />} />
          <Route path={ROUTES.VAULT} element={<VaultPage />} />
          <Route path={ROUTES.KNOWLEDGE_GRAPH} element={<KnowledgeGraphPage />} />

          {/* Operator-only surfaces — kept out of the nav; a typed URL bounces to Overview */}
          {HIDDEN_ROUTES.map((path) => (
            <Route key={path} path={path} element={<Navigate to={ROUTES.OVERVIEW} replace />} />
          ))}
        </Route>

        {/* Catch-all → overview */}
        <Route path="*" element={<Navigate to={ROUTES.OVERVIEW} replace />} />
      </Routes>
    </Suspense>
    </ErrorBoundary>
  );
}
