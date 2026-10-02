import { describe, expect, it } from "vitest";

// The console ships en + es. When a locale is added, add it here and to
// SUPPORTED_LANGUAGES in lib/constants.ts.
const LOCALES = ["en"] as const;

describe("traces i18n filter keys", () => {
  it("all trace locales expose the same top-level keys", async () => {
    const en = await import("@/i18n/locales/en/traces.json");
    const enKeys = Object.keys(en as Record<string, unknown>).sort();

    for (const locale of LOCALES) {
      const mod = await import(`@/i18n/locales/${locale}/traces.json`);
      expect(Object.keys(mod as Record<string, unknown>).sort(), locale).toEqual(enKeys);
    }
  });

  it("filter labels exist in all locales", async () => {
    for (const locale of LOCALES) {
      const mod = await import(`@/i18n/locales/${locale}/traces.json`);
      const data = mod as unknown as { filters?: Record<string, string> };
      for (const key of ["search", "status", "advanced", "from", "to", "toolName", "hasToolCalls", "clearAll"]) {
        expect(data.filters?.[key], `${locale} missing filters.${key}`).toBeTruthy();
      }
    }
  });
});
