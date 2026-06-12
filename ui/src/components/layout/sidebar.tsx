import { Link } from "@tanstack/react-router";
import {
  Activity,
  ArrowLeftRight,
  BarChart3,
  Bell,
  Bot,
  Brain,
  ClipboardCheck,
  CreditCard,
  FileText,
  Flag,
  Gauge,
  Globe,
  Key,
  KeyRound,
  LayoutDashboard,
  Link2,
  MessageSquare,
  Network,
  Route,
  ScrollText,
  Server,
  Settings,
  ShieldAlert,
  SlidersHorizontal,
  Tag,
  Terminal,
  Users,
  Waypoints,
} from "lucide-react";
import { useSettingsStore } from "@/stores/settings";

export const NAV_ITEMS = [
  { label: "Dashboard", to: "/dashboard", icon: LayoutDashboard },
  { label: "Providers", to: "/providers", icon: Server },
  { label: "Connections", to: "/connections", icon: Link2 },
  { label: "Combos", to: "/combos", icon: Activity },
  { label: "Usage", to: "/usage", icon: BarChart3 },
  { label: "Logs", to: "/logs", icon: ScrollText },
  { label: "Traffic", to: "/traffic", icon: Activity },
  { label: "Quota", to: "/quota", icon: Gauge },
  { label: "Pricing", to: "/pricing", icon: CreditCard },
  { label: "Virtual Keys", to: "/virtual-keys", icon: KeyRound },
  { label: "Routing Rules", to: "/routing-rules", icon: Route },
  { label: "Model Limits", to: "/model-limits", icon: SlidersHorizontal },
  { label: "Aliases", to: "/aliases", icon: Tag },
  { label: "Teams", to: "/teams", icon: Users },
  { label: "Audit", to: "/audit", icon: ClipboardCheck },
  { label: "Feature Flags", to: "/feature-flags", icon: Flag },
  { label: "Guardrails", to: "/guardrails", icon: ShieldAlert },
  { label: "Prompts", to: "/prompts", icon: FileText },
  { label: "Alerts", to: "/alerts", icon: Bell },
  { label: "MCP", to: "/mcp", icon: Bot },
  { label: "Skills", to: "/skills", icon: Brain },
  { label: "Settings", to: "/settings", icon: Settings },
  { label: "Keys", to: "/keys", icon: Key },
  { label: "Endpoint", to: "/endpoint", icon: Globe },
  { label: "Tunnels", to: "/tunnels", icon: Waypoints },
  { label: "MITM", to: "/mitm", icon: ArrowLeftRight },
  { label: "Proxy Pools", to: "/proxy-pools", icon: Network },
  { label: "Chat", to: "/chat", icon: MessageSquare },
  { label: "Console", to: "/console", icon: Terminal },
];

const linkClass =
  "flex items-center gap-3 px-3 py-2 rounded-md text-sm text-muted-foreground hover:bg-muted transition-colors";
const activeLinkClass =
  "flex items-center gap-3 px-3 py-2 rounded-md text-sm bg-primary/10 text-primary font-medium";

export function Sidebar() {
  const updateAvailable = useSettingsStore((state) => state.updateAvailable);
  const latestVersion = useSettingsStore((state) => state.latestVersion);

  return (
    <aside
      data-testid="desktop-sidebar"
      className="hidden lg:flex flex-col w-64 h-screen border-r border-border bg-background"
    >
      <div data-testid="traffic-lights" className="flex items-center gap-2 px-4 py-3">
        <span className="w-3 h-3 rounded-full bg-red-500" />
        <span className="w-3 h-3 rounded-full bg-yellow-500" />
        <span className="w-3 h-3 rounded-full bg-green-500" />
      </div>

      <div className="px-4 pb-3 font-bold text-lg text-foreground">g0router</div>

      <nav className="flex-1 overflow-y-auto px-2">
        <ul className="space-y-1">
          {NAV_ITEMS.map((item) => (
            <li key={item.to}>
              <Link to={item.to} className={linkClass} activeProps={{ className: activeLinkClass }}>
                <item.icon className="w-4 h-4 shrink-0" />
                <span>{item.label}</span>
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      {updateAvailable && latestVersion ? (
        <div
          data-testid="update-badge"
          className="mx-3 mb-3 px-3 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium"
        >
          {latestVersion}
        </div>
      ) : null}
    </aside>
  );
}
