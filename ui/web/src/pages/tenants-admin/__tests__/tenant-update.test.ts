import { describe, expect, it } from "vitest";
import { BYTES_PER_GIB } from "@/lib/disk-quota";
import type { TenantData } from "@/types/tenant";
import { buildUpdateParams, capIsSet } from "../tenant-update";

function tenant(overrides: Partial<TenantData> = {}): TenantData {
  return {
    id: "0198c9ab-0000-7000-8000-000000000001",
    name: "  Acme  ",
    slug: "acme",
    status: "active",
    settings: undefined,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("capIsSet", () => {
  it("is false for a legacy row with no settings and true once a cap exists", () => {
    expect(capIsSet(tenant())).toBe(false);
    expect(capIsSet(tenant({ settings: {} }))).toBe(false);
    expect(capIsSet(tenant({ settings: { disk_quota_bytes: BYTES_PER_GIB } }))).toBe(true);
  });
});

describe("buildUpdateParams", () => {
  it("trims the name and includes a cap only when the row has none", () => {
    const result = buildUpdateParams(tenant(), "  Acme  ", "20");
    expect(result).toEqual({
      params: {
        id: "0198c9ab-0000-7000-8000-000000000001",
        name: "Acme",
        settings: { disk_quota_bytes: 20 * BYTES_PER_GIB },
      },
    });
  });

  it("validates the cap against the create dialog's rules", () => {
    expect(buildUpdateParams(tenant(), "Acme", "0")).toEqual({ error: "outOfRange" });
    expect(buildUpdateParams(tenant(), "Acme", "0.5")).toEqual({ error: "notWhole" });
    expect(buildUpdateParams(tenant(), "Acme", "")).toEqual({ error: "required" });
  });

  it("never sends a cap for a row that already has one, even with a stale input", () => {
    // A broken form would forward the (refused) cap and dead-end on the API.
    const capped = tenant({ settings: { disk_quota_bytes: BYTES_PER_GIB } });
    const result = buildUpdateParams(capped, "Renamed", "999999");
    expect(result).toEqual({
      params: { id: "0198c9ab-0000-7000-8000-000000000001", name: "Renamed" },
    });
  });

  it("requires a name", () => {
    expect(buildUpdateParams(tenant(), "   ", "20")).toEqual({ error: "nameRequired" });
  });
});
