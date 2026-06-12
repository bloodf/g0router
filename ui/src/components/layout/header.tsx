import { useLocation } from "@tanstack/react-router";
import { Menu, User } from "lucide-react";
import { useHeaderSearchStore } from "@/stores/header-search";
import { useUserStore } from "@/stores/user";

function formatTitle(segment: string) {
  return segment
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

function usePageTitle() {
  const location = useLocation();
  const segment = location.pathname.split("/").filter(Boolean)[0] ?? "dashboard";
  return formatTitle(segment);
}

function ThemeToggleSlot() {
  return <span data-testid="theme-toggle-slot" />;
}

function LanguageSwitcherSlot() {
  return <span data-testid="language-switcher-slot" />;
}

function LogoutSlot() {
  return <span data-testid="logout-slot" />;
}

interface HeaderProps {
  onMenuClick: () => void;
}

export function Header({ onMenuClick }: HeaderProps) {
  const title = usePageTitle();
  const query = useHeaderSearchStore((state) => state.query);
  const setQuery = useHeaderSearchStore((state) => state.setQuery);
  const user = useUserStore((state) => state.user);

  return (
    <header className="flex items-center gap-4 px-4 py-3 border-b border-border bg-background">
      <button
        type="button"
        data-testid="mobile-hamburger"
        onClick={onMenuClick}
        className="lg:hidden p-2 rounded-md hover:bg-muted"
        aria-label="Open sidebar"
      >
        <Menu className="w-5 h-5 text-foreground" />
      </button>

      <div className="flex-1 min-w-0">
        <div className="text-xs text-muted-foreground">Home / {title}</div>
        <h1 className="text-lg font-semibold text-foreground truncate">{title}</h1>
      </div>

      <input
        type="search"
        placeholder="Search..."
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        className="hidden sm:block w-48 md:w-64 px-3 py-1.5 rounded-md border border-border bg-background text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
      />

      <div className="flex items-center gap-2 shrink-0">
        {user ? (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-md border border-border bg-muted">
            <User className="w-4 h-4 text-foreground" />
            <span className="text-sm text-foreground">{user.username}</span>
          </div>
        ) : null}
        <ThemeToggleSlot />
        <LanguageSwitcherSlot />
        <LogoutSlot />
      </div>
    </header>
  );
}
