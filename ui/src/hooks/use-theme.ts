import { useCallback, useSyncExternalStore } from "react";
import { useThemeStore, type Theme } from "@/stores/theme";

function subscribe(callback: () => void) {
  const mql = window.matchMedia("(prefers-color-scheme: dark)");
  mql.addEventListener("change", callback);
  return () => mql.removeEventListener("change", callback);
}

function getSnapshot() {
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

export function useTheme() {
  const theme = useThemeStore((state) => state.theme);
  const setTheme = useThemeStore((state) => state.setTheme);
  const systemPrefersDark = useSyncExternalStore(subscribe, getSnapshot);

  const resolvedTheme: "light" | "dark" =
    theme === "system" ? (systemPrefersDark ? "dark" : "light") : theme;

  return {
    theme,
    resolvedTheme,
    setTheme: useCallback((value: Theme) => setTheme(value), [setTheme]),
  };
}
