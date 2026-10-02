import { ArrowRight, History } from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/shared/empty-state";
import { useSessions } from "@/pages/sessions/hooks/use-sessions";
import { formatRelativeTime } from "@/lib/format";
import { ROUTES } from "@/lib/constants";

/** How many sessions the overview previews. */
const PREVIEW_LIMIT = 5;

/**
 * The last few conversations, on the landing page — "pick up where I left off"
 * without a trip to the Sessions list. `useSessions` is react-query keyed, so
 * passing the limit keeps this cache entry separate from the full list page.
 */
export function RecentSessionsCard() {
  const { t } = useTranslation("overview");
  const { sessions } = useSessions({ limit: PREVIEW_LIMIT });

  return (
    <Card className="gap-4">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <History className="h-4 w-4 text-muted-foreground" />
          {t("recentSessions.title")}
        </CardTitle>
        {sessions.length > 0 && (
          <Link
            to={ROUTES.SESSIONS}
            className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("recentSessions.viewAll")} <ArrowRight className="h-3 w-3" />
          </Link>
        )}
      </CardHeader>
      <CardContent>
        {sessions.length === 0 ? (
          <EmptyState icon={History} title={t("recentSessions.empty")} className="py-10" />
        ) : (
          <div className="space-y-3">
            {sessions.slice(0, PREVIEW_LIMIT).map((session) => (
              <Link
                key={session.key}
                to={`/sessions/${encodeURIComponent(session.key)}`}
                className="flex items-center gap-3 text-sm transition-colors hover:text-foreground"
              >
                <span className="min-w-0 flex-1 truncate">{session.label || session.key}</span>
                {session.agentName && (
                  <span className="shrink-0 text-xs text-muted-foreground">{session.agentName}</span>
                )}
                <span className="shrink-0 text-xs text-muted-foreground">
                  {t("recentSessions.messages", { count: session.messageCount })}
                </span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {formatRelativeTime(session.updated)}
                </span>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
