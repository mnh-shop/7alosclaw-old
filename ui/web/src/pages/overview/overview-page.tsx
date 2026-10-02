import { useEffect, useCallback } from "react";
import { Activity, Hash, Radio } from "lucide-react";
import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/shared/page-header";
import { StatusBadge } from "@/components/shared/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuthStore } from "@/stores/use-auth-store";
import { useWsCall } from "@/hooks/use-ws-call";
import { useWsEvent } from "@/hooks/use-ws-event";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { useProviders } from "@/pages/providers/hooks/use-providers";
import { useTraces } from "@/pages/traces/hooks/use-traces";
import { Methods, Events } from "@/api/protocol";
import { formatTokens } from "@/lib/format";

import type {
  HealthPayload,
  StatusPayload,
  QuotaUsageResult,
  CronListPayload,
  ChannelStatusPayload,
} from "./types";
import { useLiveUptime } from "./hooks/use-live-uptime";
import { StatCard } from "./stat-card";
import { useOverviewSparklines } from "./hooks/use-overview-sparklines";
import { SystemHealthCard } from "./system-health-card";
import { ConnectedClientsCard } from "./connected-clients-card";
import { CronJobsCard } from "./cron-jobs-card";
import { RecentRequestsCard } from "./recent-requests-card";
import { QuotaUsageCard } from "./quota-usage-card";
import { ApprovalsCard } from "./approvals-card";
import { RecentSessionsCard } from "./recent-sessions-card";
import { useRuntimes } from "@/pages/skills/hooks/use-runtimes";
import {
  getChannelAttentionPriority,
  getChannelStatusFallback,
} from "@/pages/channels/channels-status-view";
import { useChannelInstances } from "@/pages/channels/hooks/use-channel-instances";

const REFRESH_INTERVAL = 30_000;
const MAX_OVERVIEW_CHANNEL_INSTANCES = 200;

/** Shown before the gateway reports anything, so the page never looks half-built. */
function WaitingForGateway() {
  const { t } = useTranslation("overview");
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border bg-card px-6 py-16 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
        <Radio className="h-5 w-5 text-muted-foreground" />
      </div>
      <p className="text-sm font-medium">{t("common:connecting")}</p>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">
        {t("common:disconnectedGateway")}
      </p>
    </div>
  );
}

/** Placeholder for the first load, matching the real metric + status layout. */
function OverviewSkeleton() {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <div key={i} className="rounded-xl border bg-card p-5 shadow-sm sm:p-6">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="mt-3 h-8 w-32" />
            <Skeleton className="mt-5 h-10 w-full" />
          </div>
        ))}
      </div>
      <div className="rounded-xl border bg-card p-5 shadow-sm sm:p-6">
        <Skeleton className="h-4 w-36" />
        <div className="mt-5 grid grid-cols-2 gap-x-6 gap-y-5 sm:grid-cols-3 lg:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="space-y-2">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-4 w-24" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export function OverviewPage() {
  const { t } = useTranslation("overview");
  const connected = useAuthStore((s) => s.connected);
  const { call: fetchHealth, data: health, loading: healthLoading } =
    useWsCall<HealthPayload>(Methods.HEALTH);
  const { call: fetchStatus, data: status } =
    useWsCall<StatusPayload>(Methods.STATUS);
  const { call: fetchQuota, data: quota } =
    useWsCall<QuotaUsageResult>(Methods.QUOTA_USAGE);
  const sparklines = useOverviewSparklines();
  const { call: fetchCron, data: cronData } =
    useWsCall<CronListPayload>(Methods.CRON_LIST);
  const { call: fetchChannels, data: channelStatusData } =
    useWsCall<ChannelStatusPayload>(Methods.CHANNELS_STATUS);
  const { providers } = useProviders();
  const { runtimes } = useRuntimes();
  const { traces } = useTraces({ limit: 8 });
  const { instances: channelInstances, total: channelInstanceTotal } = useChannelInstances({
    limit: MAX_OVERVIEW_CHANNEL_INSTANCES,
    offset: 0,
  });

  const fetchAll = useCallback(() => {
    fetchHealth();
    fetchStatus();
    fetchQuota();
    fetchCron({ includeDisabled: true });
    fetchChannels();
  }, [fetchHealth, fetchStatus, fetchQuota, fetchCron, fetchChannels]);

  useEffect(() => {
    if (!connected) return;
    fetchAll();
    const id = setInterval(fetchAll, REFRESH_INTERVAL);
    return () => clearInterval(id);
  }, [connected, fetchAll]);

  const handleHealthEvent = useCallback(() => {
    fetchHealth();
    fetchStatus();
  }, [fetchHealth, fetchStatus]);
  useWsEvent(Events.HEALTH, handleHealthEvent);

  const liveUptime = useLiveUptime(health?.uptime);

  // Computed
  const agents = status?.agents ?? [];
  const runningAgents = agents.filter((a) => a.isRunning).length;
  const agentTotal = status?.agentTotal ?? agents.length;
  const channelStatusMap = channelStatusData?.channels ?? {};
  const canSynthesizeOverviewFallbacks =
    channelInstanceTotal > 0 &&
    channelInstanceTotal <= MAX_OVERVIEW_CHANNEL_INSTANCES &&
    channelInstances.length >= channelInstanceTotal;
  const channelEntries = (() => {
    const combined = new Map(Object.entries(channelStatusMap));
    if (canSynthesizeOverviewFallbacks) {
      for (const instance of channelInstances) {
        if (combined.has(instance.name)) continue;
        const fallback = getChannelStatusFallback(instance);
        if (fallback) {
          combined.set(instance.name, fallback);
        }
      }
    }
    return [...combined.entries()];
  })();
  const totalChannelCount = Math.max(channelEntries.length, channelInstanceTotal);
  const channelsOnline = channelEntries.filter(([, c]) => c.running).length;
  const channelsNeedingAttention = channelEntries.filter(
    ([, c]) => getChannelAttentionPriority(c, c.enabled) > 0,
  ).length;
  const overviewAttentionCount = canSynthesizeOverviewFallbacks
    ? channelsNeedingAttention
    : null;
  const enabledProviders = providers.filter((p) => p.enabled);
  const clientList = health?.clients ?? [];

  // Only the first load gets a skeleton; later refreshes keep the last values.
  const showOverviewSkeleton = useDeferredLoading(
    connected && !health && healthLoading,
  );

  return (
    <div className="space-y-6 p-4 sm:p-6">
      {/* Header */}
      <PageHeader
        title={t("title")}
        description={t("description")}
        actions={
          <div className="flex items-center gap-2">
            {health?.version && (
              <span className="text-xs text-muted-foreground">
                {health.version}
              </span>
            )}
            {health?.updateAvailable && health.latestVersion && (
              <a
                href={health.updateUrl || "#"}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 rounded-md border border-info/25 bg-info/10 px-2 py-0.5 text-xs font-medium text-info transition-colors hover:bg-info/20"
              >
                {health.latestVersion} available
              </a>
            )}
            <StatusBadge
              status={connected ? "success" : "error"}
              label={connected ? t("common:connected", "Connected") : t("common:disconnected", "Disconnected")}
            />
          </div>
        }
      />

      <div className="space-y-6">
      {!connected && !health ? (
        <WaitingForGateway />
      ) : showOverviewSkeleton ? (
        <OverviewSkeleton />
      ) : (
        <>
          {/* Headline metrics — today's traffic. Cost lives under Monitoring > Usage. */}
          <div className="grid gap-4 sm:grid-cols-2">
            <StatCard
              icon={Activity}
              label={t("statCards.requestsToday")}
              value={quota?.requestsToday ?? 0}
              sub={
                quota?.uniqueUsersToday
                  ? t("statCards.users", { count: quota.uniqueUsersToday })
                  : undefined
              }
              sparkline={sparklines?.requestSparkline}
              trend={sparklines?.trends.requests}
            />
            <StatCard
              icon={Hash}
              label={t("statCards.tokensToday")}
              value={formatTokens(
                (quota?.inputTokensToday ?? 0) + (quota?.outputTokensToday ?? 0),
              )}
              sub={
                quota
                  ? t("statCards.inOut", { input: formatTokens(quota.inputTokensToday), output: formatTokens(quota.outputTokensToday) })
                  : undefined
              }
              sparkline={sparklines?.tokenSparkline}
              trend={sparklines?.trends.tokens}
            />
          </div>

          {/* Runtime status — agents and channels live here, not as flat counters */}
          <SystemHealthCard
            health={health}
            liveUptime={liveUptime}
            enabledProviderCount={enabledProviders.length}
            sessions={status?.sessions ?? 0}
            clientCount={clientList.length}
            channelEntries={channelEntries}
            runtimeEntries={runtimes?.runtimes}
            agentRunning={runningAgents}
            agentTotal={agentTotal}
            channelOnline={channelsOnline}
            channelTotal={totalChannelCount}
            channelsNeedingAttention={overviewAttentionCount}
          />

          {/* What needs you, and where you left off — ApprovalsCard self-hides for
              viewers, since approvals are an operator surface. */}
          <div className="grid gap-4 lg:grid-cols-2">
            <ApprovalsCard />
            <RecentSessionsCard />
          </div>

          {/* Secondary detail */}
          <div className="grid gap-4 lg:grid-cols-2">
            <ConnectedClientsCard
              clients={clientList}
              currentId={health?.currentId}
            />
            <CronJobsCard jobs={cronData?.jobs ?? []} />
          </div>

          <RecentRequestsCard traces={traces} />

          {quota?.enabled && quota.entries.length > 0 && (
            <QuotaUsageCard quota={quota} />
          )}
        </>
      )}
      </div>
    </div>
  );
}
