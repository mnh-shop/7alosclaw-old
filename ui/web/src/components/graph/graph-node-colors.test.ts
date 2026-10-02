import { describe, expect, it } from "vitest";
import { KG_DEFAULT_COLOR, KG_TYPE_COLORS } from "@/adapters/kg-graph.adapter";
import { resolveTokenColor } from "./graph-node-colors";

describe("resolveTokenColor", () => {
  it("passes non-var colours through unchanged", () => {
    expect(resolveTokenColor("#E85D24")).toBe("#E85D24");
  });

  it("falls back to the raw var() string when the token cannot be resolved", () => {
    // jsdom applies no stylesheets, so the custom property reads empty.
    expect(resolveTokenColor("var(--brand-accent)")).toBe("var(--brand-accent)");
  });
});

describe("KG type colours", () => {
  it("carry only token references — no raw hex in the adapter", () => {
    for (const color of Object.values(KG_TYPE_COLORS)) {
      expect(color).toMatch(/^var\(--/);
    }
    expect(KG_DEFAULT_COLOR).toMatch(/^var\(--/);
  });
});
