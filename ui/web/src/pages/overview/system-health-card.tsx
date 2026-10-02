import { Monitor, CheckCircle2, XCircle, Minus, Tag } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { HealthPayload, ChannelStatusEntry } from "./types";
import type { RuntimeInfo } from "@/pages/skills/hooks/use-runtimes";
import { formatUptime } from "./hooks/use-live-uptime";
import { cleanVersion } from "@/lib/clean-version";
import { getChannelAttentionPriority, getChannelStatusMeta } from "@/pages/channels/channels-status-view";
import { ChannelAttentionPanel } from "./channel-attention-panel";

function StatusDot({ ok }: { ok: boolean | undefined }) {
  if (ok === undefined)
    return <Minus className="h-3.5 w-3.5 text-muted-foreground/40" />;
  return ok ? (
    <CheckCircle2 className="h-3.5 w-3.5 text-success" />
  ) : (
    <XCircle className="h-3.5 w-3.5 text-destructive" />
  );
}

function HealthFact({
  label,
  value,
  statusOk,
}: {
  label: string;
  value: string;
  statusOk?: boolean;
}) {
  return (
    <div className="min-w-0">
      <p className="brand-eyebrow text-muted-foreground">{label}</p>
      <div className="mt-1.5 flex items-center gap-1.5">
        {statusOk !== undefined && <StatusDot ok={statusOk} />}
        <p className="text-sm font-medium break-words tabular-nums">{value}</p>
      </div>
    </div>
  );
}

export function SystemHealthCard({
  health,
  liveUptime,
  enabledProviderCount,
  sessions,
  clientCount,
  channelEntries,
  runtimeEntries,
  agentRunning,
  agentTotal,
  channelOnline,
  channelTotal,
  channelsNeedingAttention,
}: {
  health: HealthPayload | null;
  liveUptime: number | undefined;
  enabledProviderCount: number;
  sessions: number;
  clientCount: number;
  channelEntries: [string, ChannelStatusEntry][];
  runtimeEntries?: RuntimeInfo[];
  agentRunning: number;
  agentTotal: number;
  channelOnline: number;
  channelTotal: number;
  channelsNeedingAttention: number | null;
}) {
  const { t } = useTranslation("overview");
  const degradedCount = channelEntries.filter(
    ([, ch]) => ch.state === "degraded",
  ).length;
  const failedCount = channelEntries.filter(
    ([, ch]) => ch.state === "failed",
  ).length;
  const attentionEntries = [...channelEntries]
    .filter(([, ch]) => getChannelAttentionPriority(ch, ch.enabled) > 0)
    .sort(
      (a, b) =>
        getChannelAttentionPriority(b[1], b[1].enabled) -
        getChannelAttentionPriority(a[1], a[1].enabled),
    );

  return (
    <Card className="gap-5">
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <Monitor className="h-4 w-4 text-muted-foreground" />
            {t("systemHealth.title")}
          </CardTitle>
          {health?.version && (
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Tag className="h-3 w-3" />
              <span className="font-medium">
                {cleanVersion(health.version)}
              </span>
              {health.updateAvailable === false && (
                <CheckCircle2 className="h-3 w-3 text-success" />
              )}
              {health.updateAvailable && health.latestVersion && (
                <span className="ml-1 rounded-md bg-warning/15 px-2 py-0.5 text-xs font-medium text-warning">
                  {health.latestVersion} available
                </span>
              )}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid grid-cols-2 gap-x-6 gap-y-5 sm:grid-cols-3 lg:grid-cols-4">
          <HealthFact
            label={t("systemHealth.uptime")}
            value={formatUptime(liveUptime)}
          />
          {health?.database && (
            <HealthFact
              label={t("systemHealth.database")}
              value={
                health.database === "ok"
                  ? t("common:connected", "Connected")
                  : health.database
              }
              statusOk={health.database === "ok"}
            />
          )}
          <HealthFact
            label={t("systemHealth.providers")}
            value={
              enabledProviderCount > 0
                ? t("systemHealth.active", { count: enabledProviderCount })
                : t("systemHealth.none")
            }
            statusOk={enabledProviderCount > 0}
          />
          <HealthFact
            label={t("statCards.agents")}
            value={agentTotal > 0 ? `${agentRunning} / ${agentTotal}` : "0"}
          />
          <HealthFact
            label={t("systemHealth.channels")}
            value={
              channelTotal > 0 ? `${channelOnline} / ${channelTotal}` : "0"
            }
            statusOk={
              channelsNeedingAttention == null
                ? undefined
                : channelsNeedingAttention === 0
            }
          />
          <HealthFact
            label={t("systemHealth.sessions")}
            value={String(sessions)}
          />
          <HealthFact
            label={t("systemHealth.clients")}
            value={String(clientCount)}
          />
          <HealthFact
            label={t("systemHealth.tools")}
            value={String(health?.tools ?? 0)}
          />
        </div>

        {runtimeEntries && runtimeEntries.length > 0 && (
          <div className="border-t pt-5">
            <p className="brand-eyebrow mb-3 text-muted-foreground">
              {t("systemHealth.runtimes")}
            </p>
            <div className="flex flex-wrap gap-1.5">
              {runtimeEntries.map((rt) => (
                <span
                  key={rt.name}
                  className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2 py-1 text-xs"
                >
                  <span
                    className={`h-1.5 w-1.5 rounded-full ${rt.available ? "bg-success" : "bg-destructive"}`}
                  />
                  {rt.name}
                  {rt.version && (
                    <span className="text-muted-foreground">{rt.version}</span>
                  )}
                </span>
              ))}
            </div>
          </div>
        )}

        {channelEntries.length > 0 && (
          <div className="border-t pt-5">
            <div className="mb-3 flex items-center justify-between gap-2">
              <p className="brand-eyebrow text-muted-foreground">
                {t("systemHealth.channels")}
              </p>
              {(degradedCount > 0 || failedCount > 0) && (
                <span className="text-xs text-muted-foreground">
                  {failedCount > 0
                    ? t("systemHealth.failedCount", {
                        defaultValue: "{{count}} failed",
                        count: failedCount,
                      })
                    : degradedCount === 1
                      ? t("systemHealth.warningCountOne", {
                          defaultValue: "{{count}} warning",
                          count: degradedCount,
                        })
                      : t("systemHealth.warningCountOther", {
                          defaultValue: "{{count}} warnings",
                          count: degradedCount,
                        })}
                </span>
              )}
            </div>
            <div className="flex flex-wrap gap-1.5">
              {channelEntries.map(([name, ch]) => {
                const meta = getChannelStatusMeta(ch, ch.enabled, t);
                return (
                  <span
                    key={name}
                    className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2 py-1 text-xs"
                  >
                    <span
                      className={`h-1.5 w-1.5 rounded-full ${meta.dotClass}`}
                    />
                    {name}
                  </span>
                );
              })}
            </div>
            <ChannelAttentionPanel attentionEntries={attentionEntries} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
