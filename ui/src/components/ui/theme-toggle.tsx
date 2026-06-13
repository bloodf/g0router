import { Monitor, Moon, Sun } from "lucide-react";

import { useThemeStore, type Theme } from "@/stores/theme";
import { cn } from "@/lib/utils";

export function nextTheme(theme: Theme): Theme {
  switch (theme) {
    case "light":
      return "dark";
    case "dark":
      return "system";
    case "system":
    default:
      return "light";
  }
}

const icons: Record<Theme, typeof Sun> = {
  light: Sun,
  dark: Moon,
  system: Monitor,
};

export interface ThemeToggleProps {
  className?: string;
}

function ThemeToggle({ className }: ThemeToggleProps) {
  const theme = useThemeStore((state) => state.theme);
  const setTheme = useThemeStore((state) => state.setTheme);
  const Icon = icons[theme];

  return (
    <button
      type="button"
      aria-label={`Theme: ${theme}`}
      onClick={() => setTheme(nextTheme(theme))}
      className={cn(
        "inline-flex size-8 items-center justify-center rounded-md border border-border text-foreground transition-colors hover:bg-muted",
        className
      )}
    >
      <Icon className="size-4" />
    </button>
  );
}

export { ThemeToggle };
