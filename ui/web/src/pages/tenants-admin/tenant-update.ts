// Pure logic for the tenant edit dialog, so it can be tested without a server.
// Mirrors internal/gateway/methods/tenants.go handleUpdate: name applies when
// non-empty, and the settings patch may set disk_quota_bytes only once when the
// stored row has none (the engine refuses changes to an existing cap).

import { parseDiskQuotaGiB, type DiskQuotaError } from "@/lib/disk-quota";
import type { TenantData } from "@/types/tenant";

export type TenantUpdateError = "nameRequired" | DiskQuotaError;

// The index signature matches the wire type ws.call expects (Record<string, unknown>).
export interface TenantUpdateParams {
  [key: string]: unknown;
  id: string;
  name: string;
  settings?: { disk_quota_bytes: number };
}

/** True when the stored row already carries a disk cap (post-cap tenant). */
export function capIsSet(tenant: TenantData): boolean {
  return typeof tenant.settings?.disk_quota_bytes === "number";
}

/** Build the tenants.update params, or name the rejection. */
export function buildUpdateParams(
  tenant: TenantData,
  name: string,
  quotaGiB: string,
): { params: TenantUpdateParams } | { error: TenantUpdateError } {
  const trimmed = name.trim();
  if (!trimmed) return { error: "nameRequired" };

  const params: TenantUpdateParams = { id: tenant.id, name: trimmed };

  // The cap is offered only when the row has none; when it is set, quotaGiB is
  // ignored outright (the field is not rendered) so a stale value can never
  // reach an API that will refuse it.
  if (!capIsSet(tenant)) {
    const parsed = parseDiskQuotaGiB(quotaGiB);
    if ("error" in parsed) return { error: parsed.error };
    params.settings = { disk_quota_bytes: parsed.bytes };
  }

  return { params };
}
