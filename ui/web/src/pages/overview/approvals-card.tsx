import { ArrowRight, ShieldCheck } from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/shared/empty-state";
import { useCanWrite } from "@/components/shared/require-role";
import { useApprovals } from "@/pages/approvals/hooks/use-approvals";
import { formatRelativeTime } from "@/lib/format";
import { ROUTES } from "@/lib/constants";

const PREVIEW_LIMIT = 5;

/**
 * The operator action queue, on the landing page: approvals are the one thing on
 * this console that is both urgent and blocking, so they belong where you arrive
 * rather than one navigation away.
 *
 * Self-gating — approvals are an operator surface, and a read-only viewer should
 * see nothing here rather than a control the engine would reject.
 */
export function ApprovalsCard() {
  const { t } = useTranslation("overview");
  const canWrite = useCanWrite();
  const { pending } = useApprovals();

  if (!canWrite) return null;

  return (
    <Card className="gap-4">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <ShieldCheck className="h-4 w-4 text-muted-foreground" />
          {t("approvals.title")}
          {pending.length > 0 && (
            <span className="rounded-md bg-warning/15 px-1.5 py-0.5 text-2xs font-medium text-warning">
              {t("approvals.waiting", { count: pending.length })}
            </span>
          )}
        </CardTitle>
        {pending.length > 0 && (
          <Link
            to={ROUTES.APPROVALS}
            className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("approvals.review")} <ArrowRight className="h-3 w-3" />
          </Link>
        )}
      </CardHeader>
      <CardContent>
        {pending.length === 0 ? (
          <EmptyState icon={ShieldCheck} title={t("approvals.empty")} className="py-10" />
        ) : (
          <div className="space-y-3">
            {pending.slice(0, PREVIEW_LIMIT).map((approval) => (
              <div key={approval.id} className="flex items-center gap-3 text-sm">
                <span className="min-w-0 flex-1 truncate font-mono text-xs">{approval.command}</span>
                <span className="shrink-0 text-xs text-muted-foreground">{approval.agentId}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {formatRelativeTime(new Date(approval.createdAt))}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
