// Tenant disk cap: operators think in GiB, the engine takes whole bytes.
// Mirror of the engine ceiling in internal/config/tenant_workspace.go
// (MaxTenantDiskQuotaBytes = 1 << 40 = 1 TiB).

export const BYTES_PER_GIB = 1_073_741_824;
export const MAX_TENANT_DISK_QUOTA_BYTES = 1_099_511_627_776; // 1 TiB
export const MAX_DISK_QUOTA_GIB = MAX_TENANT_DISK_QUOTA_BYTES / BYTES_PER_GIB; // 1024
export const DEFAULT_DISK_QUOTA_GIB = 10;

export type DiskQuotaError = "required" | "notWhole" | "outOfRange";

/** Parse an operator-entered GiB cap into engine bytes, or name the rejection. */
export function parseDiskQuotaGiB(
  input: string,
): { bytes: number } | { error: DiskQuotaError } {
  const raw = input.trim();
  if (raw === "") return { error: "required" };
  const gib = Number(raw);
  if (!Number.isFinite(gib) || !Number.isInteger(gib)) return { error: "notWhole" };
  if (gib < 1 || gib > MAX_DISK_QUOTA_GIB) return { error: "outOfRange" };
  return { bytes: gib * BYTES_PER_GIB };
}
