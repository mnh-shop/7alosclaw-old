import { useEffect } from "react";

/**
 * The dashboard is dark only. `index.html` already ships `class="dark"` on
 * <html> so the first paint is dark; this just re-asserts it (and clears any
 * stale `light` class) once React mounts. There is no theme toggle.
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    const root = document.documentElement;
    root.classList.remove("light");
    root.classList.add("dark");
  }, []);

  return <>{children}</>;
}
