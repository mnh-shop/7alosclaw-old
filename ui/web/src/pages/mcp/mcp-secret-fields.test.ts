import { describe, expect, it } from "vitest";
import { buildSecretFields, includeSecretField } from "./mcp-secret-fields";

describe("MCP write-only secret fields", () => {
  it("omits untouched secret fields when editing (redacted response cannot clobber)", () => {
    const fields = buildSecretFields({
      isEditing: true,
      isStdio: false,
      headers: {},
      env: {},
      headersDirty: false,
      envDirty: false,
      headersCleared: false,
      envCleared: false,
    });
    expect(fields).toEqual({});
    expect("headers" in fields).toBe(false);
    expect("env" in fields).toBe(false);
  });

  it("sends only the field the user changed while editing", () => {
    const fields = buildSecretFields({
      isEditing: true,
      isStdio: false,
      headers: { Authorization: "Bearer new" },
      env: {},
      headersDirty: true,
      envDirty: false,
      headersCleared: false,
      envCleared: false,
    });
    expect(fields).toEqual({ headers: { Authorization: "Bearer new" } });
    expect("env" in fields).toBe(false);
  });

  it("sends an empty map only when the user explicitly clears it", () => {
    const fields = buildSecretFields({
      isEditing: true,
      isStdio: false,
      headers: {},
      env: {},
      headersDirty: false,
      envDirty: false,
      headersCleared: true,
      envCleared: false,
    });
    expect(fields).toEqual({ headers: {} });
  });

  it("always sends entered values on create", () => {
    const fields = buildSecretFields({
      isEditing: false,
      isStdio: false,
      headers: {},
      env: { TOKEN: "abc" },
      headersDirty: false,
      envDirty: false,
      headersCleared: false,
      envCleared: false,
    });
    expect(fields).toEqual({ headers: {}, env: { TOKEN: "abc" } });
  });

  it("never sends headers for stdio servers", () => {
    expect(includeSecretField({ isEditing: true, dirty: true, cleared: false })).toBe(true);
    const fields = buildSecretFields({
      isEditing: true,
      isStdio: true,
      headers: { "X-Test": "1" },
      env: { FOO: "bar" },
      headersDirty: true,
      envDirty: true,
      headersCleared: false,
      envCleared: false,
    });
    expect(fields).toEqual({ env: { FOO: "bar" } });
  });
});
