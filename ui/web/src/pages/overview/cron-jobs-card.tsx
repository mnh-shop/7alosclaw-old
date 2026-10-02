import { ArrowRight, Timer } from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/shared/empty-state";
import { ROUTES } from "@/lib/constants";
import { formatRelativeTime } from "@/lib/format";
import type { CronJob } from "./types";

export function CronJobsCard({ jobs }: { jobs: CronJob[] }) {
  const { t } = useTranslation("overview");
  return (
    <Card className="gap-4">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Timer className="h-4 w-4 text-muted-foreground" />
          {t("cronJobs.title")}
        </CardTitle>
        {jobs.length > 0 && (
          <Link
            to={ROUTES.CRON}
            className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("cronJobs.manage")} <ArrowRight className="h-3 w-3" />
          </Link>
        )}
      </CardHeader>
      <CardContent>
        {jobs.length === 0 ? (
          <EmptyState
            icon={Timer}
            title={t("cronJobs.noJobs")}
            className="py-10"
          />
        ) : (
          <div className="space-y-3">
            {jobs.slice(0, 5).map((job) => (
              <div
                key={job.id}
                className="flex items-center justify-between gap-3 text-sm"
              >
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className={`h-1.5 w-1.5 shrink-0 rounded-full ${
                      job.enabled ? "bg-success" : "bg-muted-foreground/40"
                    }`}
                  />
                  <Link
                    to={`/cron/${job.id}`}
                    className={`truncate hover:underline ${
                      job.enabled ? "" : "text-muted-foreground"
                    }`}
                  >
                    {job.name}
                  </Link>
                </div>
                <span className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
                  {job.enabled && job.state.nextRunAtMs ? (
                    <>
                      <Timer className="h-3 w-3" />
                      {formatRelativeTime(
                        new Date(job.state.nextRunAtMs),
                      ).replace(" ago", "")}
                    </>
                  ) : !job.enabled ? (
                    t("cronJobs.disabled")
                  ) : (
                    "--"
                  )}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
