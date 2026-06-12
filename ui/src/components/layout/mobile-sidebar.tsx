import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { NAV_ITEMS } from "./sidebar";

interface MobileSidebarProps {
  open: boolean;
  onClose: () => void;
}

export function MobileSidebar({ open, onClose }: MobileSidebarProps) {
  return (
    <>
      <div
        data-testid="mobile-sidebar-overlay"
        onClick={onClose}
        className={cn(
          "fixed inset-0 z-40 bg-black/50 lg:hidden transition-opacity",
          open ? "opacity-100" : "opacity-0 pointer-events-none"
        )}
      />
      <aside
        data-testid="mobile-sidebar"
        className={cn(
          "fixed top-0 left-0 z-50 w-64 h-full bg-background border-r border-border lg:hidden transform transition-transform duration-200 ease-in-out",
          open ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <div className="px-4 py-3 font-bold text-lg text-foreground">g0router</div>
        <nav className="px-2">
          <ul className="space-y-1">
            {NAV_ITEMS.map((item) => {
              const Icon = item.icon;
              return (
                <li key={item.to}>
                  <Link
                    to={item.to}
                    onClick={onClose}
                    className="flex items-center gap-3 px-3 py-2 rounded-md text-sm text-muted-foreground hover:bg-muted transition-colors"
                    activeProps={{
                      className:
                        "flex items-center gap-3 px-3 py-2 rounded-md text-sm bg-primary/10 text-primary font-medium",
                    }}
                  >
                    <Icon className="w-4 h-4 shrink-0" />
                    <span>{item.label}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>
      </aside>
    </>
  );
}
