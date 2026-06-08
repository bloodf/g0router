import { Icon } from "../common/Icon";
import { useTheme } from "@/lib/theme";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { useTranslation } from "react-i18next";
import { useAuthStatus, useLogout } from "@/lib/auth";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { getNavSections } from "./Sidebar";

function Breadcrumbs() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const { t } = useTranslation();
  const sections = getNavSections(t);
  const current = sections
    .flatMap((s) => s.items)
    .find(
      (i) =>
        pathname === i.href ||
        (i.href !== "/dashboard" && pathname?.startsWith(i.href)),
    );

  return (
    <div className="flex items-center gap-2 text-sm">
      {current && (
        <>
          <Icon name={current.icon} size={18} className="text-text-muted" />
          <span className="font-semibold">{t(current.labelKey)}</span>
        </>
      )}
      {!current && <span className="font-semibold">g0router</span>}
    </div>
  );
}

export function Header({ onMenuClick }: { onMenuClick?: () => void }) {
  const { theme, resolved, cycle } = useTheme();
  const { i18n } = useTranslation();
  const { data: status } = useAuthStatus();
  const logout = useLogout();
  const navigate = useNavigate();

  const themeIcon =
    theme === "light" ? "light_mode" : theme === "dark" ? "dark_mode" : "computer";

  return (
    <header className="h-16 border-b border-border bg-background/80 backdrop-blur-md flex items-center justify-between px-4 md:px-6 sticky top-0 z-30">
      <div className="flex items-center gap-3 min-w-0">
        <Button
          variant="ghost"
          size="icon"
          className="md:hidden"
          onClick={onMenuClick}
          aria-label="Open menu"
        >
          <Icon name="menu" />
        </Button>
        <Breadcrumbs />
      </div>

      <div className="flex items-center gap-1">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="gap-1.5">
              <Icon name="translate" size={16} />
              <span className="hidden md:inline text-xs uppercase" suppressHydrationWarning>
                {i18n.language?.startsWith("pt") ? "PT" : "EN"}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => i18n.changeLanguage("en")}>
              🇺🇸 English
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => i18n.changeLanguage("pt-BR")}>
              🇧🇷 Português (BR)
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <Button
          variant="ghost"
          size="icon"
          onClick={cycle}
          aria-label={`Theme: ${theme} (${resolved})`}
          title={`Theme: ${theme}`}
        >
          <Icon name={themeIcon} />
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="gap-2" data-testid="user-menu">
              <div className="w-7 h-7 rounded-full bg-gradient-to-br from-brand-400 to-brand-600 flex items-center justify-center text-white text-xs font-semibold">
                {status?.display_name?.[0]?.toUpperCase() ?? "A"}
              </div>
              <span className="hidden md:inline text-sm">
                {status?.display_name ?? "admin"}
              </span>
              <Icon name="expand_more" size={16} />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuLabel>{status?.username ?? "—"}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => navigate({ to: "/settings" })}>
              <Icon name="settings" size={16} className="mr-2" />
              Settings
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={async () => {
                await logout.mutateAsync();
                navigate({ to: "/login" });
              }}
            >
              <Icon name="logout" size={16} className="mr-2" />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
