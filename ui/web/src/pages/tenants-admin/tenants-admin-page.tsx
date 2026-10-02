import { useState } from "react";
import { useNavigate } from "react-router";
import { useTranslation } from "react-i18next";
import { Plus, RefreshCw, Building2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { useMinLoading } from "@/hooks/use-min-loading";
import { useTenantsAdmin } from "./hooks/use-tenants-admin";
import { ROUTES } from "@/lib/constants";
import {
  DEFAULT_DISK_QUOTA_GIB,
  MAX_DISK_QUOTA_GIB,
  parseDiskQuotaGiB,
  type DiskQuotaError,
} from "@/lib/disk-quota";

function statusVariant(status: string): "default" | "secondary" | "destructive" {
  if (status === "active") return "default";
  if (status === "suspended") return "destructive";
  return "secondary";
}

function diskQuotaErrorKey(err: DiskQuotaError): string {
  if (err === "required") return "diskQuotaRequired";
  if (err === "notWhole") return "diskQuotaWhole";
  return "diskQuotaRange";
}

export function TenantsAdminPage() {
  const { t } = useTranslation("tenants");
  const { t: tc } = useTranslation("common");
  const navigate = useNavigate();
  const { tenants, loading, refreshing, refresh, createTenant, isOwner } = useTenantsAdmin();

  const spinning = useMinLoading(refreshing);
  const showSkeleton = useDeferredLoading(loading && tenants.length === 0);

  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [diskQuota, setDiskQuota] = useState(String(DEFAULT_DISK_QUOTA_GIB));
  const [diskQuotaError, setDiskQuotaError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  const parsedQuota = parseDiskQuotaGiB(diskQuota);
  const quotaBytes = "bytes" in parsedQuota ? parsedQuota.bytes : null;

  const handleCreate = async () => {
    if (!name.trim() || !slug.trim()) return;
    const parsed = parseDiskQuotaGiB(diskQuota);
    if ("error" in parsed) {
      setDiskQuotaError(t(diskQuotaErrorKey(parsed.error)));
      return;
    }
    setCreating(true);
    setSubmitError(null);
    try {
      await createTenant({
        name: name.trim(),
        slug: slug.trim(),
        diskQuotaBytes: parsed.bytes,
      });
      setCreateOpen(false);
      setName("");
      setSlug("");
      setDiskQuota(String(DEFAULT_DISK_QUOTA_GIB));
      setDiskQuotaError(null);
    } catch (err) {
      // The engine's rejection already names the key and the range, so show it verbatim.
      setSubmitError(err instanceof Error ? err.message : t("createError"));
    } finally {
      setCreating(false);
    }
  };

  const handleDiskQuotaChange = (v: string) => {
    setDiskQuota(v);
    const parsed = parseDiskQuotaGiB(v);
    setDiskQuotaError("error" in parsed ? t(diskQuotaErrorKey(parsed.error)) : null);
  };

  const handleNameChange = (v: string) => {
    setName(v);
    // Auto-derive slug from name if not manually edited
    setSlug(v.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, ""));
  };

  return (
    <div className="p-4 sm:p-6 pb-10">
      <PageHeader
        title={t("title")}
        description={t("description")}
        actions={
          <div className="flex gap-2">
            {isOwner && (
              <Button
                size="sm"
                onClick={() => {
                  setSubmitError(null);
                  setDiskQuotaError(null);
                  setDiskQuota(String(DEFAULT_DISK_QUOTA_GIB));
                  setCreateOpen(true);
                }}
                className="gap-1"
              >
                <Plus className="h-3.5 w-3.5" /> {t("createTenant")}
              </Button>
            )}
            <Button variant="outline" size="sm" onClick={refresh} disabled={spinning} className="gap-1">
              <RefreshCw className={spinning ? "animate-spin h-3.5 w-3.5" : "h-3.5 w-3.5"} />
              {tc("refresh")}
            </Button>
          </div>
        }
      />

      <div className="mt-4">
        {showSkeleton ? (
          <TableSkeleton rows={5} />
        ) : tenants.length === 0 ? (
          <EmptyState icon={Building2} title={t("noTenants")} description={t("description")} />
        ) : (
          <div className="rounded-md border overflow-x-auto">
            <table className="w-full min-w-[600px] text-base md:text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-2 text-left font-medium">{t("name")}</th>
                  <th className="px-4 py-2 text-left font-medium">{t("slug")}</th>
                  <th className="px-4 py-2 text-left font-medium">{t("status")}</th>
                  <th className="px-4 py-2 text-left font-medium">{t("created")}</th>
                </tr>
              </thead>
              <tbody>
                {tenants.map((tenant) => (
                  <tr
                    key={tenant.id}
                    className="border-b last:border-0 hover:bg-muted/30 cursor-pointer"
                    onClick={() => navigate(ROUTES.TENANT_DETAIL.replace(":id", tenant.id))}
                  >
                    <td className="px-4 py-2 font-medium">{tenant.name}</td>
                    <td className="px-4 py-2">
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{tenant.slug}</code>
                    </td>
                    <td className="px-4 py-2">
                      <Badge variant={statusVariant(tenant.status)}>
                        {t(tenant.status) || tenant.status}
                      </Badge>
                    </td>
                    <td className="px-4 py-2 text-muted-foreground text-xs">
                      {new Date(tenant.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-sm:inset-0 max-sm:translate-x-0 max-sm:translate-y-0 sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("createTenant")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="tenant-name">{t("name")}</Label>
              <Input
                id="tenant-name"
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder={t("name")}
                className="text-base md:text-sm"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="tenant-slug">{t("slug")}</Label>
              <Input
                id="tenant-slug"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
                placeholder="my-org"
                className="text-base md:text-sm"
              />
              <p className="text-xs text-muted-foreground">{t("slugHelp")}</p>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="tenant-disk-quota">{t("diskQuota")}</Label>
              <Input
                id="tenant-disk-quota"
                type="number"
                inputMode="numeric"
                min={1}
                max={MAX_DISK_QUOTA_GIB}
                step={1}
                value={diskQuota}
                onChange={(e) => handleDiskQuotaChange(e.target.value)}
                placeholder={String(DEFAULT_DISK_QUOTA_GIB)}
                className="text-base md:text-sm"
                aria-invalid={!!diskQuotaError}
              />
              <p className="text-xs text-muted-foreground">{t("diskQuotaHelp")}</p>
              {quotaBytes !== null && !diskQuotaError && (
                <p className="text-xs text-muted-foreground">
                  {t("diskQuotaBytes", { bytes: quotaBytes.toLocaleString() })}
                </p>
              )}
              {diskQuotaError && <p className="text-xs text-destructive">{diskQuotaError}</p>}
            </div>
          </div>
          {submitError && <p className="text-sm text-destructive">{submitError}</p>}
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)} disabled={creating}>
              {tc("cancel")}
            </Button>
            <Button
              onClick={handleCreate}
              disabled={creating || !name.trim() || !slug.trim() || !!diskQuotaError}
            >
              {t("createTenant")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
