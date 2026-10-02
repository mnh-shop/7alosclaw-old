import { describe, expect, it } from "vitest";
import { resolveGraphNodeDisplayColor } from "./graph-node-colors";

describe("resolveGraphNodeDisplayColor", () => {
  it("keeps the adapter-assigned color for knowledge graph entity nodes", () => {
    expect(resolveGraphNodeDisplayColor({
      entityType: "person",
      color: "#E85D24",
    })).toBe("#E85D24");
  });

  it("uses the dark vault colors for vault document nodes", () => {
    expect(resolveGraphNodeDisplayColor({
      docType: "document",
      entityType: "document",
      color: "#8b5cf6",
    })).toBe("#22d3ee");
  });
});
