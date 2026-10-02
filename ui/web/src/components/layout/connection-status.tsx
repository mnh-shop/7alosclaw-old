import { useTranslation } from "react-i18next";
import { useAuthStore } from "@/stores/use-auth-store";
import { cn } from "@/lib/utils";
import { cleanVersion } from "@/lib/clean-version";

/** Role chips, exported because the Options page reuses them. This console has
 *  no red/amber/green any more, so the four roles are separated by brand hue and
 *  weight instead: owner violet (highest), admin cyan, operator the neutral
 *  slate, viewer muted. Admin must NOT use `destructive` — with the status
 *  palette now violet, that would collide with owner and read as an error. */
export const ROLE_STYLES: Record<string, string> = {
  owner: "bg-secondary/15 text-brand-secondary",
  admin: "bg-primary/15 text-primary",
  operator: "bg-info/15 text-info",
  viewer: "bg-muted text-muted-foreground",
};

export function ConnectionStatus({ collapsed }: { collapsed?: boolean }) {
  const { t } = useTranslation("common");
  const connected = useAuthStore((s) => s.connected);
  const serverVersion = useAuthStore((s) => s.serverInfo?.version);
  const tenantName = useAuthStore((s) => s.tenantName);
  const role = useAuthStore((s) => s.role);

  return (
    <div className="space-y-1.5">
      {/* Tenant + role (expanded only) */}
      {!collapsed && tenantName && (
        <div className="flex items-center justify-between gap-1.5 text-xs overflow-hidden">
          <span className="truncate font-medium text-foreground/80">{tenantName}</span>
          {role && (
            <span className={cn("shrink-0 rounded-md px-1.5 py-0.5 text-2xs font-medium", ROLE_STYLES[role] ?? ROLE_STYLES.viewer)}>
              {role}
            </span>
          )}
        </div>
      )}

      {/* Connection status */}
      <div className="flex items-center gap-2 text-xs text-muted-foreground overflow-hidden">
        <span
          className={cn(
            "h-2 w-2 shrink-0 rounded-full",
            connected ? "bg-success" : "bg-destructive",
          )}
        />
        {!collapsed && (
          <span className="truncate">
            {connected ? t("connected") : t("disconnected")}
            {connected && serverVersion && (
              <span className="ml-1 opacity-60">· {cleanVersion(serverVersion)}</span>
            )}
          </span>
        )}
      </div>
    </div>
  );
}
