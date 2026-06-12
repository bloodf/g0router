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
          <li>
            <Link to="/dashboard" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <LayoutDashboard className="w-4 h-4 shrink-0" />
              <span>Dashboard</span>
            </Link>
          </li>
          <li>
            <Link to="/providers" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Server className="w-4 h-4 shrink-0" />
              <span>Providers</span>
            </Link>
          </li>
          <li>
            <Link to="/connections" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Link2 className="w-4 h-4 shrink-0" />
              <span>Connections</span>
            </Link>
          </li>
          <li>
            <Link to="/combos" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Activity className="w-4 h-4 shrink-0" />
              <span>Combos</span>
            </Link>
          </li>
          <li>
            <Link to="/usage" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <BarChart3 className="w-4 h-4 shrink-0" />
              <span>Usage</span>
            </Link>
          </li>
          <li>
            <Link to="/logs" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <ScrollText className="w-4 h-4 shrink-0" />
              <span>Logs</span>
            </Link>
          </li>
          <li>
            <Link to="/traffic" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Activity className="w-4 h-4 shrink-0" />
              <span>Traffic</span>
            </Link>
          </li>
          <li>
            <Link to="/quota" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Gauge className="w-4 h-4 shrink-0" />
              <span>Quota</span>
            </Link>
          </li>
          <li>
            <Link to="/pricing" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <CreditCard className="w-4 h-4 shrink-0" />
              <span>Pricing</span>
            </Link>
          </li>
          <li>
            <Link to="/virtual-keys" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <KeyRound className="w-4 h-4 shrink-0" />
              <span>Virtual Keys</span>
            </Link>
          </li>
          <li>
            <Link to="/routing-rules" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Route className="w-4 h-4 shrink-0" />
              <span>Routing Rules</span>
            </Link>
          </li>
          <li>
            <Link to="/model-limits" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <SlidersHorizontal className="w-4 h-4 shrink-0" />
              <span>Model Limits</span>
            </Link>
          </li>
          <li>
            <Link to="/aliases" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Tag className="w-4 h-4 shrink-0" />
              <span>Aliases</span>
            </Link>
          </li>
          <li>
            <Link to="/teams" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Users className="w-4 h-4 shrink-0" />
              <span>Teams</span>
            </Link>
          </li>
          <li>
            <Link to="/audit" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <ClipboardCheck className="w-4 h-4 shrink-0" />
              <span>Audit</span>
            </Link>
          </li>
          <li>
            <Link to="/feature-flags" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Flag className="w-4 h-4 shrink-0" />
              <span>Feature Flags</span>
            </Link>
          </li>
          <li>
            <Link to="/guardrails" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <ShieldAlert className="w-4 h-4 shrink-0" />
              <span>Guardrails</span>
            </Link>
          </li>
          <li>
            <Link to="/prompts" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <FileText className="w-4 h-4 shrink-0" />
              <span>Prompts</span>
            </Link>
          </li>
          <li>
            <Link to="/alerts" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Bell className="w-4 h-4 shrink-0" />
              <span>Alerts</span>
            </Link>
          </li>
          <li>
            <Link to="/mcp" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Bot className="w-4 h-4 shrink-0" />
              <span>MCP</span>
            </Link>
          </li>
          <li>
            <Link to="/skills" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Brain className="w-4 h-4 shrink-0" />
              <span>Skills</span>
            </Link>
          </li>
          <li>
            <Link to="/settings" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Settings className="w-4 h-4 shrink-0" />
              <span>Settings</span>
            </Link>
          </li>
          <li>
            <Link to="/keys" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Key className="w-4 h-4 shrink-0" />
              <span>Keys</span>
            </Link>
          </li>
          <li>
            <Link to="/endpoint" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Globe className="w-4 h-4 shrink-0" />
              <span>Endpoint</span>
            </Link>
          </li>
          <li>
            <Link to="/tunnels" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Waypoints className="w-4 h-4 shrink-0" />
              <span>Tunnels</span>
            </Link>
          </li>
          <li>
            <Link to="/mitm" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <ArrowLeftRight className="w-4 h-4 shrink-0" />
              <span>MITM</span>
            </Link>
          </li>
          <li>
            <Link to="/proxy-pools" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Network className="w-4 h-4 shrink-0" />
              <span>Proxy Pools</span>
            </Link>
          </li>
          <li>
            <Link to="/chat" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <MessageSquare className="w-4 h-4 shrink-0" />
              <span>Chat</span>
            </Link>
          </li>
          <li>
            <Link to="/console" className={linkClass} activeProps={{ className: activeLinkClass }}>
              <Terminal className="w-4 h-4 shrink-0" />
              <span>Console</span>
            </Link>
          </li>
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
