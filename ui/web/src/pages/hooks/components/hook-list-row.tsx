import { Webhook, Pencil, Trash2, FlaskConical } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import type { HookConfig } from "@/hooks/use-hooks";
import { useCanWrite } from "@/components/shared/require-role";
import { SystemBadge } from "./system-badge";

interface HookListRowProps {
  hook: HookConfig;
  onClick?: () => void;
  onToggle: (enabled: boolean) => void;
  onEdit: () => void;
  onDelete: () => void;
  onTest: () => void;
}

const EVENT_COLORS: Record<string, string> = {
  session_start: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300",
  user_prompt_submit: "bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300",
  pre_tool_use: "bg-warning text-warning dark:bg-warning/30 dark:text-warning",
  post_tool_use: "bg-success text-success dark:bg-success/30 dark:text-success",
  stop: "bg-destructive text-destructive dark:bg-destructive/30 dark:text-destructive",
  subagent_start: "bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300",
  subagent_stop: "bg-warning text-warning dark:bg-warning/30 dark:text-warning",
};

const HANDLER_COLORS: Record<string, string> = {
  command: "bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300",
  http: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300",
  prompt: "bg-destructive text-destructive dark:bg-destructive/30 dark:text-destructive",
  script: "bg-success text-success dark:bg-success/30 dark:text-success",
};

export function HookListRow({ hook, onClick, onToggle, onEdit, onDelete, onTest }: HookListRowProps) {
  const { t } = useTranslation("hooks");
  const canWrite = useCanWrite();

  return (
    <div className="flex w-full items-center gap-3 rounded-lg border bg-card px-4 py-3 transition-all hover:border-primary/30 hover:shadow-sm">
      {/* Icon */}
      <button type="button" onClick={onClick} className="flex h-8 w-8 shrink-0 cursor-pointer items-center justify-center rounded-lg bg-primary/10 text-primary">
        <Webhook className="h-4 w-4" />
      </button>

      {/* Main info */}
      <button type="button" onClick={onClick} className="min-w-0 flex-1 text-left">
        {hook.name && (
          <p className="truncate text-sm font-medium">{hook.name}</p>
        )}
        <div className={`flex flex-wrap items-center gap-1.5 ${hook.name ? "mt-0.5" : ""}`}>
          <span
            className={`inline-flex items-center rounded px-1.5 py-0.5 text-2xs font-medium ${EVENT_COLORS[hook.event] ?? "bg-muted text-muted-foreground"}`}
          >
            {hook.event}
          </span>
          <span
            className={`inline-flex items-center rounded px-1.5 py-0.5 text-2xs font-medium ${HANDLER_COLORS[hook.handler_type] ?? "bg-muted text-muted-foreground"}`}
          >
            {hook.handler_type}
          </span>
          <Badge variant="outline" className="text-2xs px-1 py-0">
            {hook.scope}
          </Badge>
          {hook.source === "builtin" && <SystemBadge />}
          {hook.scope === "agent" && hook.agent_ids && hook.agent_ids.length > 0 && (
            <Badge variant="outline" className="text-2xs px-1 py-0">
              {hook.agent_ids.length} agent{hook.agent_ids.length > 1 ? "s" : ""}
            </Badge>
          )}
        </div>
        {hook.matcher && (
          <p className="mt-0.5 truncate text-xs text-muted-foreground font-mono">{hook.matcher}</p>
        )}
      </button>

      {/* Toggle */}
      {canWrite && (
        <div className="shrink-0" onClick={(e) => e.stopPropagation()}>
          <Switch
            checked={hook.enabled}
            onCheckedChange={onToggle}
            aria-label={t("actions.toggle")}
            className="scale-90"
          />
        </div>
      )}

      {/* Actions */}
      {canWrite && (
        <div className="flex shrink-0 items-center gap-1" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="xs"
            className="text-muted-foreground hover:text-primary"
            onClick={onTest}
            title={t("actions.test")}
          >
            <FlaskConical className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="xs"
            className="text-muted-foreground hover:text-primary"
            onClick={onEdit}
            title={t("actions.edit")}
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="xs"
            className="text-muted-foreground hover:text-destructive disabled:opacity-40"
            onClick={onDelete}
            disabled={hook.source === "builtin"}
            title={hook.source === "builtin" ? t("form.builtinReadonly") : t("actions.delete")}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      )}
    </div>
  );
}
