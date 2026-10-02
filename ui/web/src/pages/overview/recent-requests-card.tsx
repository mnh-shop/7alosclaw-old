import { useMemo } from "react";
import { Activity, ArrowRight } from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { StatusBadge } from "@/components/shared/status-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/shared/empty-state";
import { ROUTES } from "@/lib/constants";
import {
  formatRelativeTime,
  formatTokens,
  formatDuration,
} from "@/lib/format";
import { useContactResolver } from "@/hooks/use-contact-resolver";
import { formatUserLabel } from "@/lib/format-user-label";

interface Trace {
  id: string;
  name: string;
  user_id: string;
  channel: string;
  total_input_tokens: number;
  total_output_tokens: number;
  duration_ms: number;
  status: string;
  created_at: string;
}

export function RecentRequestsCard({ traces }: { traces: Trace[] }) {
  const { t } = useTranslation("overview");
  const userIds = useMemo(() => traces.map((tr) => tr.user_id).filter(Boolean) as string[], [traces]);
  const { resolve } = useContactResolver(userIds);
  return (
    <Card className="gap-4">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Activity className="h-4 w-4 text-muted-foreground" />
          {t("recentRequests.title")}
        </CardTitle>
        {traces.length > 0 && (
          <Link
            to={ROUTES.TRACES}
            className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("recentRequests.viewAll")} <ArrowRight className="h-3 w-3" />
          </Link>
        )}
      </CardHeader>
      <CardContent>
        {traces.length === 0 ? (
          <EmptyState
            icon={Activity}
            title={t("recentRequests.noRequests")}
            className="py-10"
          />
        ) : (
          <div className="overflow-x-auto overscroll-contain">
            <table className="w-full min-w-[720px] text-sm">
              <thead>
                <tr className="border-b text-left text-muted-foreground">
                  <th className="pb-2 pr-4 font-medium whitespace-nowrap">{t("recentRequests.columns.time")}</th>
                  <th className="pb-2 px-4 font-medium">{t("recentRequests.columns.name")}</th>
                  <th className="pb-2 px-4 font-medium">{t("recentRequests.columns.user")}</th>
                  <th className="pb-2 px-4 font-medium">{t("recentRequests.columns.channel")}</th>
                  <th className="pb-2 px-4 text-right font-medium whitespace-nowrap">{t("recentRequests.columns.tokens")}</th>
                  <th className="pb-2 px-4 text-right font-medium whitespace-nowrap">
                    {t("recentRequests.columns.duration")}
                  </th>
                  <th className="pb-2 pl-4 font-medium">{t("recentRequests.columns.status")}</th>
                </tr>
              </thead>
              <tbody>
                {traces.map((trace) => (
                  <tr key={trace.id} className="border-b last:border-0">
                    <td className="py-2.5 pr-4 text-muted-foreground whitespace-nowrap">
                      {formatRelativeTime(trace.created_at)}
                    </td>
                    <td className="py-2.5 px-4 max-w-[160px] truncate">
                      {trace.name || "--"}
                    </td>
                    <td className="py-2.5 px-4 font-mono text-xs">
                      {trace.user_id ? formatUserLabel(trace.user_id, resolve) : "--"}
                    </td>
                    <td className="py-2.5 px-4">{trace.channel || "--"}</td>
                    <td className="py-2.5 px-4 text-right tabular-nums">
                      {formatTokens(
                        trace.total_input_tokens + trace.total_output_tokens,
                      )}
                    </td>
                    <td className="py-2.5 px-4 text-right tabular-nums">
                      {formatDuration(trace.duration_ms)}
                    </td>
                    <td className="py-2.5 pl-4">
                      <StatusBadge
                        status={
                          trace.status === "completed"
                            ? "success"
                            : trace.status === "error"
                              ? "error"
                              : trace.status === "running"
                                ? "info"
                                : "default"
                        }
                        label={trace.status}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
