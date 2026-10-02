import { defineConfig } from "vitest/config";
import path from "path";

export default defineConfig({
  test: {
    // jsdom provides browser globals: crypto.subtle, Blob, File, TextEncoder
    environment: "jsdom",
    globals: false,
    // Restores localStorage/sessionStorage, which a Node >= 22 global shadows
    // out from under jsdom. The file itself explains why.
    setupFiles: ["./vitest.setup.ts"],
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
