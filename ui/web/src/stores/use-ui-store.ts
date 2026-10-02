import { create } from "zustand";
import { persist } from "zustand/middleware";

/** The dashboard is dark only. Kept as a union so read sites stay explicit;
 *  the store pins it to "dark" and never offers a setter. */
export type Theme = "light" | "dark" | "system";

interface UiState {
  theme: Theme;
  timezone: string; // IANA timezone or "auto"
  sidebarCollapsed: boolean;
  mobileSidebarOpen: boolean;
  pageSize: number; // global pagination page size preference

  setTimezone: (tz: string) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setMobileSidebarOpen: (open: boolean) => void;
  setPageSize: (size: number) => void;
}

export const useUiStore = create<UiState>()(
  persist(
    (set, get) => ({
      theme: "dark" as Theme,
      timezone: "auto",
      sidebarCollapsed: false,
      mobileSidebarOpen: false,
      pageSize: 20,

      setTimezone: (tz) => {
        set({ timezone: tz });
      },

      toggleSidebar: () => {
        set({ sidebarCollapsed: !get().sidebarCollapsed });
      },

      setSidebarCollapsed: (collapsed) => {
        set({ sidebarCollapsed: collapsed });
      },

      setMobileSidebarOpen: (open) => set({ mobileSidebarOpen: open }),

      setPageSize: (size) => set({ pageSize: size }),
    }),
    {
      name: "zalosclaw:ui", // localStorage key
      partialize: (state) => ({
        // Persist user preferences — not transient UI state
        theme: state.theme,
        timezone: state.timezone,
        sidebarCollapsed: state.sidebarCollapsed,
        pageSize: state.pageSize,
      }),
      // A "light"/"system" value persisted by an older build must not leak
      // through and leave the app half-lit — force the only supported theme.
      merge: (persisted, current) => ({
        ...current,
        ...(persisted as Partial<UiState>),
        theme: "dark",
      }),
    }
  )
);
