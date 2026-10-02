/**
 * jsdom installs `localStorage` / `sessionStorage`, but Node >= 22 also defines
 * `globalThis.localStorage` — an experimental own accessor that resolves to
 * `undefined` unless node is started with `--localstorage-file`. It shadows
 * jsdom's, so any module-scope `localStorage.getItem(...)` in the app throws
 * `Cannot read properties of undefined` at import time (see
 * `src/i18n/index.ts` and `src/stores/use-auth-store.ts`, which both read it
 * while the module is being evaluated).
 *
 * Defining a real Storage here keeps the suite independent of the Node version.
 */
function createStorage(): Storage {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => values.delete(key),
    setItem: (key, value) => values.set(key, String(value)),
  };
}

Object.defineProperty(window, "localStorage", {
  configurable: true,
  value: createStorage(),
});
Object.defineProperty(window, "sessionStorage", {
  configurable: true,
  value: createStorage(),
});
