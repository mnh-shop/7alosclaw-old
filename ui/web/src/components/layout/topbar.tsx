import { PanelLeftClose, PanelLeftOpen, Menu } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useUiStore } from "@/stores/use-ui-store";
import { useIsMobile } from "@/hooks/use-media-query";

/**
 * The topbar holds the sidebar toggle and nothing else.
 *
 * It used to carry a gear (Options shortcut) and an account menu (tenant,
 * tenants link, API keys, about, logout) on the right. The owner asked for that
 * to be gone from the upper area, so: everything that was in the account menu
 * now lives where it belongs — the per-user settings, tenant switcher and
 * account details are on the Options page, reached via the sidebar's
 * System → Settings folder, and the account actions (about, logout) moved onto
 * that page's Account card. The embedding-status dot that hung off the gear is
 * already shown on the Options page's System Settings card.
 *
 * Keep it that way: a persistent top-right control cluster is exactly what was
 * removed here, so do not add one back without asking.
 */
export function Topbar() {
  const { t } = useTranslation("topbar");
  const sidebarCollapsed = useUiStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useUiStore((s) => s.toggleSidebar);
  const setMobileSidebarOpen = useUiStore((s) => s.setMobileSidebarOpen);
  const isMobile = useIsMobile();

  const handleSidebarToggle = isMobile
    ? () => setMobileSidebarOpen(true)
    : toggleSidebar;

  const toggleLabel = isMobile
    ? t("openMenu")
    : sidebarCollapsed
      ? t("expandSidebar")
      : t("collapseSidebar");

  return (
    <header className="flex h-14 items-center border-b bg-background px-4 landscape-compact">
      <button
        onClick={handleSidebarToggle}
        className="cursor-pointer rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
        title={toggleLabel}
        aria-label={toggleLabel}
      >
        {isMobile ? (
          <Menu className="h-4 w-4" />
        ) : sidebarCollapsed ? (
          <PanelLeftOpen className="h-4 w-4" />
        ) : (
          <PanelLeftClose className="h-4 w-4" />
        )}
      </button>
    </header>
  );
}
