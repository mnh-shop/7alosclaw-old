import { getVaultNodeColor } from "@/adapters/vault-graph-adapter";

/** Dark-only — the console has no light theme to resolve against. */
export function resolveGraphNodeDisplayColor(attrs: Record<string, unknown>): string {
  const docType = typeof attrs.docType === "string" ? attrs.docType : "";
  if (docType) {
    return getVaultNodeColor(docType);
  }
  return typeof attrs.color === "string" && attrs.color
    ? attrs.color
    : getVaultNodeColor("other");
}

/** Resolve a `var(--…)` token to its computed colour. Sigma's WebGL programs
    and the minimap's 2D canvas read raw colour strings and cannot resolve
    custom properties themselves, so graph node attributes carry the resolved
    value; CSS consumers (legend dots) can keep the `var()` form. */
export function resolveTokenColor(color: string): string {
  if (!color.startsWith("var(")) return color;
  const name = color.slice(4, -1).trim();
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || color;
}
