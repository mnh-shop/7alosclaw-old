import { describe, expect, it } from "vitest";
import {
  BYTES_PER_GIB,
  MAX_TENANT_DISK_QUOTA_BYTES,
  parseDiskQuotaGiB,
} from "./disk-quota";

describe("parseDiskQuotaGiB", () => {
  it("converts whole GiB to bytes", () => {
    expect(parseDiskQuotaGiB("10")).toEqual({ bytes: 10 * BYTES_PER_GIB });
    expect(parseDiskQuotaGiB(" 25 ")).toEqual({ bytes: 25 * BYTES_PER_GIB });
  });

  it("accepts the 1 GiB floor and the 1 TiB ceiling", () => {
    expect(parseDiskQuotaGiB("1")).toEqual({ bytes: BYTES_PER_GIB });
    expect(parseDiskQuotaGiB("1024")).toEqual({ bytes: MAX_TENANT_DISK_QUOTA_BYTES });
  });

  it("rejects empty, zero, negative, fractional and over-maximum values", () => {
    expect(parseDiskQuotaGiB("")).toEqual({ error: "required" });
    expect(parseDiskQuotaGiB("   ")).toEqual({ error: "required" });
    expect(parseDiskQuotaGiB("0")).toEqual({ error: "outOfRange" });
    expect(parseDiskQuotaGiB("-5")).toEqual({ error: "outOfRange" });
    expect(parseDiskQuotaGiB("0.5")).toEqual({ error: "notWhole" });
    expect(parseDiskQuotaGiB("abc")).toEqual({ error: "notWhole" });
    expect(parseDiskQuotaGiB("1025")).toEqual({ error: "outOfRange" });
  });
});
