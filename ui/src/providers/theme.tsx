import { useEffect } from "react";
import { initTheme, applyTheme } from "@/stores/theme";
import { useTheme } from "@/hooks/use-theme";

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    initTheme();
  }, []);

  const { theme, resolvedTheme } = useTheme();

  useEffect(() => {
    applyTheme(theme);
  }, [theme, resolvedTheme]);

  return children;
}
