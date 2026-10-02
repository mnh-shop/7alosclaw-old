import {
  LayoutDashboard,
  MessageSquare,
  Bot,
  History,
  Zap,
  Clock,
  Activity,
  BarChart3,
  Radio,
  ShieldCheck,
  Users,
  Plug,
  HardDrive,
  Inbox,
  Brain,
  Network,
  Contact,
  ArrowLeftRight,
  FileArchive,
  SlidersHorizontal,
  MonitorCog,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { SidebarGroup } from "./sidebar-group";
import { SidebarItem } from "./sidebar-item";
import { ConnectionStatus } from "./connection-status";
import { ROUTES } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/use-auth-store";
import { useCanWrite, useIsAdmin } from "@/components/shared/require-role";
import { getRuntimeBranding } from "@/lib/branding";

interface SidebarProps {
  collapsed: boolean;
  onNavItemClick?: () => void;
}

export function Sidebar({ collapsed, onNavItemClick }: SidebarProps) {
  const { t } = useTranslation("sidebar");
  const isOwner = useAuthStore((s) => s.isOwner);
  // Operator is the seat that administers and operates a customer's own tenant;
  // viewer is read-only. Admin/owner still pass, since operator is ranked below them.
  const isOperator = useCanWrite();
  // Workstations are admin-only on the backend (every method is RoleAdmin), so
  // the nav entry must be too — an operator was shown a page whose every call
  // failed. Matches the RequireAdmin guard on the route.
  const isAdmin = useIsAdmin();
  const branding = getRuntimeBranding();

  return (
    <aside
      className={cn(
        "flex h-full flex-col overflow-hidden overscroll-none border-r border-sidebar-border bg-sidebar text-sidebar-foreground backdrop-blur-xl transition-all duration-200",
        collapsed ? "w-16" : "w-64",
      )}
      onClick={(e) => {
        // Close mobile drawer when clicking a nav link
        if (onNavItemClick && (e.target as HTMLElement).closest("a")) {
          onNavItemClick();
        }
      }}
    >
      {/* Logo / title */}
      <div className="flex h-14 items-center border-b px-4">
        {!collapsed && (
          <div className="flex items-center gap-2.5">
            <img src={branding.logoUrl} alt={branding.appName} className="h-8 w-8 object-contain" />
            <span className="text-lg font-bold tracking-tight text-sidebar-primary">
              {branding.appShortName}
            </span>
          </div>
        )}
        {collapsed && (
          <img src={branding.logoUrl} alt={branding.appName} className="mx-auto h-7 w-7 object-contain" />
        )}
      </div>

      {/* Nav items */}
      <nav className="flex-1 space-y-4 overflow-y-auto overscroll-contain px-2 py-4">
        <SidebarGroup label={t("groups.core")} collapsed={collapsed}>
          <SidebarItem to={ROUTES.OVERVIEW} icon={LayoutDashboard} label={t("nav.overview")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.CHAT} icon={MessageSquare} label={t("nav.chat")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.AGENTS} icon={Bot} label={t("nav.agents")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.TEAMS} icon={Users} label={t("nav.agentTeams")} collapsed={collapsed} />
        </SidebarGroup>

        <SidebarGroup label={t("groups.conversations")} collapsed={collapsed}>
          <SidebarItem to={ROUTES.SESSIONS} icon={History} label={t("nav.sessions")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.PENDING_MESSAGES} icon={Inbox} label={t("nav.pendingMessages")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.CONTACTS} icon={Contact} label={t("nav.contacts")} collapsed={collapsed} />
        </SidebarGroup>

        <SidebarGroup label={t("groups.capabilities")} collapsed={collapsed}>
          <SidebarItem to={ROUTES.CHANNELS} icon={Radio} label={t("nav.channels")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.SKILLS} icon={Zap} label={t("nav.skills")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.MCP} icon={Plug} label={t("nav.mcpServers")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.CRON} icon={Clock} label={t("nav.cron")} collapsed={collapsed} />
          {/* Admin-only: the engine registers every workstations.* method with
              requireRole(RoleAdmin), so showing it to an operator gave a page
              that could not load. */}
          {isAdmin && (
            <SidebarItem to={ROUTES.WORKSTATIONS} icon={MonitorCog} label={t("nav.workstations")} collapsed={collapsed} />
          )}
        </SidebarGroup>

        <SidebarGroup label={t("groups.data")} collapsed={collapsed}>
          <SidebarItem to={ROUTES.MEMORY} icon={Brain} label={t("nav.memory")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.VAULT} icon={FileArchive} label={t("nav.vault")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.KNOWLEDGE_GRAPH} icon={Network} label={t("nav.knowledgeGraph")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.STORAGE} icon={HardDrive} label={t("nav.storage")} collapsed={collapsed} />
        </SidebarGroup>

        <SidebarGroup label={t("groups.monitoring")} collapsed={collapsed}>
          <SidebarItem to={ROUTES.TRACES} icon={Activity} label={t("nav.traces")} collapsed={collapsed} />
          <SidebarItem to={ROUTES.USAGE} icon={BarChart3} label={t("nav.usage")} collapsed={collapsed} />
        </SidebarGroup>

        {/* Settings replaces the old "System" group, which had become a header
            wrapping a single folder. It is a normal group like every other one
            in this sidebar — nothing else here collapses, so this must not
            either. */}
        <SidebarGroup label={t("groups.settings")} collapsed={collapsed}>
          <SidebarItem to={ROUTES.OPTIONS} icon={SlidersHorizontal} label={t("nav.options")} collapsed={collapsed} />
          {isOperator && (
            <SidebarItem to={ROUTES.APPROVALS} icon={ShieldCheck} label={t("nav.approvals")} collapsed={collapsed} />
          )}
          {/* Owner-only: canImport is isOwnerUser(), so a tenant admin would be
              shown a button the engine rejects. */}
          {isOwner && (
            <SidebarItem to={ROUTES.IMPORT_EXPORT} icon={ArrowLeftRight} label={t("nav.importExport")} collapsed={collapsed} />
          )}
        </SidebarGroup>
      </nav>

      {/* Footer: connection status */}
      <div className={cn("shrink-0 overscroll-none border-t py-3", collapsed ? "px-2 flex justify-center" : "px-4")}>
        <ConnectionStatus collapsed={collapsed} />
      </div>
    </aside>
  );
}
