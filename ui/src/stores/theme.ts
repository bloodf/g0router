import { create } from "zustand";
import { persist } from "zustand/middleware";

export type Theme = "light" | "dark" | "system";

interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

export const useThemeStore = create(
  persist<ThemeState>(
    (set) => ({
      theme: "system",
      setTheme: (theme) => {
        set({ theme });
        applyTheme(theme);
      },
    }),
    { name: "theme" }
  )
);

export function applyTheme(theme: Theme) {
  const mql =
    typeof window !== "undefined"
      ? window.matchMedia("(prefers-color-scheme: dark)")
      : globalThis.matchMedia?.("(prefers-color-scheme: dark)");
  const prefersDark = mql?.matches ?? false;
  const isDark = theme === "dark" || (theme === "system" && prefersDark);
  document.documentElement.classList.toggle("dark", isDark);
}

export function initTheme() {
  const stored = localStorage.getItem("theme");
  let theme: Theme = "system";
  if (stored) {
    try {
      const parsed = JSON.parse(stored);
      theme = parsed?.state?.theme ?? "system";
    } catch {
      theme = "system";
    }
  }
  applyTheme(theme);
}
