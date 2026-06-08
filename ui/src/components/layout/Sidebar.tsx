import { Link, useRouterState } from "@tanstack/react-router";
import { Icon } from "../common/Icon";
import { cn } from "@/lib/utils";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";

interface NavItem {
  href: string;
  labelKey: string;
  icon: string;
}

interface NavSection {
  titleKey: string;
  items: NavItem[];
}

export function getNavSections(t: (k: string) => string): NavSection[] {
  return [
    {
      titleKey: "nav.main",
      items: [
        { href: "/dashboard", labelKey: "nav.dashboard", icon: "dashboard" },
        { href: "/endpoint", labelKey: "nav.endpoint", icon: "api" },
        { href: "/providers", labelKey: "nav.providers", icon: "dns" },
        { href: "/connections", labelKey: "nav.connections", icon: "link" },
        { href: "/keys", labelKey: "nav.keys", icon: "key" },
        { href: "/virtual-keys", labelKey: "nav.virtual_keys", icon: "vpn_key" },
        { href: "/combos", labelKey: "nav.combos", icon: "layers" },
        { href: "/routing-rules", labelKey: "nav.routing", icon: "alt_route" },
        { href: "/models", labelKey: "nav.models", icon: "memory" },
        { href: "/aliases", labelKey: "nav.aliases", icon: "label" },
        { href: "/pricing", labelKey: "nav.pricing", icon: "sell" },
      ],
    },
    {
      titleKey: "nav.monitoring",
      items: [
        { href: "/usage", labelKey: "nav.usage", icon: "bar_chart" },
        { href: "/quota", labelKey: "nav.quota", icon: "data_usage" },
        { href: "/logs", labelKey: "nav.logs", icon: "description" },
        { href: "/traffic", labelKey: "nav.traffic", icon: "graph_3" },
        { href: "/console", labelKey: "nav.console", icon: "terminal" },
        { href: "/chat", labelKey: "nav.chat", icon: "chat" },
      ],
    },
    {
      titleKey: "nav.mcp",
      items: [
        { href: "/mcp", labelKey: "nav.mcp_overview", icon: "hub" },
        { href: "/mcp/instances", labelKey: "nav.mcp_instances", icon: "memory" },
        { href: "/mcp/accounts", labelKey: "nav.mcp_accounts", icon: "account_circle" },
        { href: "/mcp/tools", labelKey: "nav.mcp_tools", icon: "build" },
        { href: "/mcp/tool-groups", labelKey: "nav.mcp_tool_groups", icon: "workspaces" },
      ],
    },
    {
      titleKey: "nav.network",
      items: [
        { href: "/tunnels", labelKey: "nav.tunnels", icon: "cloud_sync" },
        { href: "/proxy-pools", labelKey: "nav.proxy_pools", icon: "lan" },
        { href: "/mitm", labelKey: "nav.mitm", icon: "security" },
      ],
    },
    {
      titleKey: "nav.advanced",
      items: [
        { href: "/teams", labelKey: "nav.teams", icon: "groups" },
        { href: "/guardrails", labelKey: "nav.guardrails", icon: "shield" },
        { href: "/model-limits", labelKey: "nav.model_limits", icon: "speed" },
        { href: "/prompts", labelKey: "nav.prompts", icon: "article" },
        { href: "/alerts", labelKey: "nav.alerts", icon: "notifications" },
        { href: "/feature-flags", labelKey: "nav.flags", icon: "flag" },
      ],
    },
    {
      titleKey: "nav.system",
      items: [
        { href: "/settings", labelKey: "nav.settings", icon: "settings" },
        { href: "/audit", labelKey: "nav.audit", icon: "history" },
        { href: "/diagnostics", labelKey: "nav.diagnostics", icon: "monitor_heart" },
        { href: "/skills", labelKey: "nav.skills", icon: "extension" },
      ],
    },
  ];
}

export function Sidebar({ onClose }: { onClose?: () => void }) {
  const { t } = useTranslation();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const sections = getNavSections(t);
  const { data: version } = useQuery({
    queryKey: ["version"],
    queryFn: () => apiFetch<{ current: string }>("/api/version"),
  });

  return (
    <aside className="bg-sidebar bg-vibrancy w-[240px] flex-shrink-0 border-r border-sidebar-border h-full flex flex-col overflow-hidden">
      <div className="h-16 flex items-center gap-2 px-5 border-b border-sidebar-border">
        <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-brand-500 to-brand-600 flex items-center justify-center text-white font-bold text-sm shadow-warm">
          g0
        </div>
        <div className="font-semibold tracking-tight">g0router</div>
      </div>

      <nav className="flex-1 overflow-y-auto custom-scrollbar py-3 px-2.5">
        {sections.map((section) => (
          <div key={section.titleKey} className="mb-4">
            <div className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-text-subtle">
              {t(section.titleKey)}
            </div>
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                const active =
                  pathname === item.href ||
                  (item.href !== "/dashboard" && pathname?.startsWith(item.href));
                return (
                  <li key={item.href}>
                    <Link
                      to={item.href}
                      onClick={onClose}
                      className={cn(
                        "flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors",
                        active
                          ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                          : "text-foreground/80 hover:bg-surface-2 hover:text-foreground",
                      )}
                    >
                      <Icon name={item.icon} size={18} />
                      <span className="truncate">{t(item.labelKey)}</span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>

      <div className="border-t border-sidebar-border px-4 py-3 flex items-center justify-between text-xs text-text-muted">
        <span>{version?.current ?? "…"}</span>
        <span className="pulse-dot" />
      </div>
    </aside>
  );
}
