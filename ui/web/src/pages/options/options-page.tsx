import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Building2, Check, Clock, Info, LogOut, Settings2, User } from "lucide-react";
import { PageHeader } from "@/components/shared/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { AboutDialog } from "@/components/layout/about-dialog";
import { useAuthStore } from "@/stores/use-auth-store";
import { useUiStore } from "@/stores/use-ui-store";
import { useTenants } from "@/hooks/use-tenants";
import { useEmbeddingStatus } from "@/hooks/use-embedding-status";
import { TIMEZONE_OPTIONS, LOCAL_STORAGE_KEYS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { ROLE_STYLES } from "@/components/layout/connection-status";
import { SystemSettingsModal } from "@/components/layout/system-settings-modal";

export function OptionsPage() {
  const { t } = useTranslation("options");
  return (
    <div className="p-4 sm:p-6 pb-10">
      <PageHeader title={t("title")} description={t("description")} />
      <div className="mt-4 grid gap-4">
        <OptionsAccountCard />
        <OptionsTimezoneCard />
        <OptionsTenantCard />
        <OptionsSystemSettingsCard />
      </div>
    </div>
  );
}

function OptionsAccountCard() {
  const { t } = useTranslation("options");
  // About/logout copy is shared with the old topbar menu, so reuse its namespace
  // rather than duplicating the strings into options.json.
  const { t: tt } = useTranslation("topbar");
  const userId = useAuthStore((s) => s.userId);
  const tenantName = useAuthStore((s) => s.tenantName);
  const role = useAuthStore((s) => s.role);
  const logout = useAuthStore((s) => s.logout);
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false);
  const [showAbout, setShowAbout] = useState(false);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <User className="h-4 w-4 text-muted-foreground" />{t("account.title")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1.5 text-sm">
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">{t("account.signedInAs")}</span>
          <span className="font-medium">{userId}</span>
          {role && (
            <span className={cn("rounded-md px-1.5 py-0.5 text-2xs font-medium", ROLE_STYLES[role] ?? ROLE_STYLES.viewer)}>
              {role}
            </span>
          )}
        </div>
        {tenantName && (
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground">{t("account.tenant")}</span>
            <span className="truncate font-medium">{tenantName}</span>
          </div>
        )}
        {/* These used to live in the topbar's account menu. */}
        <div className="flex flex-wrap gap-2 pt-3">
          <Button variant="outline" size="sm" className="gap-1.5" onClick={() => setShowAbout(true)}>
            <Info className="h-3.5 w-3.5" />
            {tt("about.menuItem")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="gap-1.5 text-destructive"
            onClick={() => setShowLogoutConfirm(true)}
          >
            <LogOut className="h-3.5 w-3.5" />
            {tt("logout")}
          </Button>
        </div>
      </CardContent>

      <ConfirmDialog
        open={showLogoutConfirm}
        onOpenChange={setShowLogoutConfirm}
        title={tt("logout")}
        description={tt("logoutConfirm")}
        confirmLabel={tt("logout")}
        variant="destructive"
        onConfirm={() => { setShowLogoutConfirm(false); logout(); }}
      />
      <AboutDialog open={showAbout} onOpenChange={setShowAbout} />
    </Card>
  );
}

function OptionsTimezoneCard() {
  const { t } = useTranslation("options");
  const timezone = useUiStore((s) => s.timezone);
  const setTimezone = useUiStore((s) => s.setTimezone);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Clock className="h-4 w-4 text-muted-foreground" />{t("timezone.title")}
        </CardTitle>
        <CardDescription>{t("timezone.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <Select value={timezone} onValueChange={setTimezone}>
          <SelectTrigger className="max-w-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {TIMEZONE_OPTIONS.map((tz) => (
              <SelectItem key={tz.value} value={tz.value}>{tz.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </CardContent>
    </Card>
  );
}

function OptionsTenantCard() {
  const { t } = useTranslation("options");
  const { t: tt } = useTranslation("tenants");
  const { tenants, isOwner, isMultiTenant, currentTenantId, currentTenantName } = useTenants();

  const handleSwitchTenant = (_tenantId: string, slug: string) => {
    // Always set TENANT_ID so WS and HTTP both resolve the correct tenant.
    localStorage.setItem(LOCAL_STORAGE_KEYS.TENANT_ID, slug);
    // Non-owner: also set TENANT_HINT for browser pairing backward compat (Path 3a).
    if (!isOwner) {
      localStorage.setItem(LOCAL_STORAGE_KEYS.TENANT_HINT, slug);
    }
    window.location.reload();
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Building2 className="h-4 w-4 text-muted-foreground" />{t("tenant.title")}
        </CardTitle>
        <CardDescription>{t("tenant.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isMultiTenant ? (
          <div className="max-h-64 space-y-0.5 overflow-y-auto">
            {tenants.map((tenant) => (
              <button
                key={tenant.id}
                onClick={() => {
                  if (tenant.id !== currentTenantId) {
                    handleSwitchTenant(tenant.id, tenant.slug);
                  }
                }}
                className="flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent"
              >
                <Building2 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                <span className="flex-1 truncate text-left">{tenant.name}</span>
                {tenant.id === currentTenantId && (
                  <>
                    <span className="text-2xs text-muted-foreground">{tt("currentTenant")}</span>
                    <Check className="h-3.5 w-3.5 shrink-0 text-primary" />
                  </>
                )}
              </button>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            {currentTenantName || t("tenant.single")}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function OptionsSystemSettingsCard() {
  const { t } = useTranslation("options");
  const role = useAuthStore((s) => s.role);
  const isAdmin = role === "admin" || role === "owner";
  const { status: embStatus } = useEmbeddingStatus();
  const [settingsOpen, setSettingsOpen] = useState(false);

  if (!isAdmin) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Settings2 className="h-4 w-4 text-muted-foreground" />{t("systemSettings.title")}
          <span
            className={cn("h-2 w-2 rounded-full", embStatus?.configured ? "bg-success" : "bg-warning")}
            title={embStatus?.configured ? undefined : t("systemSettings.open")}
          />
        </CardTitle>
        <CardDescription>{t("systemSettings.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <Button variant="outline" size="sm" onClick={() => setSettingsOpen(true)} className="gap-1.5">
          <Settings2 className="h-3.5 w-3.5" />
          {t("systemSettings.open")}
        </Button>
      </CardContent>
      <SystemSettingsModal open={settingsOpen} onOpenChange={setSettingsOpen} />
    </Card>
  );
}
