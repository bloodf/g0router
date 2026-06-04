import { type FormEvent, useState } from "react";
import "./index.css";
import { clearControlPlaneKey, getControlPlaneKey, saveControlPlaneKey } from "./api";
import { APIKeysPage } from "./pages/APIKeysPage";
import { AliasesPage } from "./pages/AliasesPage";
import { CombosPage } from "./pages/CombosPage";
import { ConnectionsAuthPage } from "./pages/ConnectionsAuthPage";
import { DashboardPage } from "./pages/DashboardPage";
import { DiagnosticsPage } from "./pages/DiagnosticsPage";
import { EndpointPage } from "./pages/EndpointPage";
import { LogsPage } from "./pages/LogsPage";
import { McpPage } from "./pages/McpPage";
import { ModelsPage } from "./pages/ModelsPage";
import { PricingPage } from "./pages/PricingPage";
import { ProvidersPage } from "./pages/ProvidersPage";
import { QuotaPage } from "./pages/QuotaPage";
import { SettingsPage } from "./pages/SettingsPage";
import { UsagePage } from "./pages/UsagePage";

const pages = [
  {
    id: "dashboard",
    label: "Dashboard",
    title: "Gateway overview",
    description: "Live operational summary for the local g0router instance.",
    Component: DashboardPage
  },
  {
    id: "endpoint",
    label: "Endpoint",
    title: "Endpoint configuration",
    description: "API keys, RTK, caveman, and request controls.",
    Component: EndpointPage
  },
  {
    id: "api-keys",
    label: "API Keys",
    title: "API Keys",
    description: "Gateway keys for authenticated client traffic.",
    Component: APIKeysPage
  },
  {
    id: "providers",
    label: "Providers",
    title: "Providers",
    description: "Connection status and provider account management.",
    Component: ProvidersPage
  },
  {
    id: "connections-auth",
    label: "Connections/Auth",
    title: "Connections/Auth",
    description: "Provider account rows, OAuth-backed sessions, API-token rows, and credential-safe actions.",
    Component: ConnectionsAuthPage
  },
  {
    id: "aliases",
    label: "Aliases",
    title: "Aliases",
    description: "Stable model names mapped to provider and upstream model targets.",
    Component: AliasesPage
  },
  {
    id: "models",
    label: "Models",
    title: "Models",
    description: "Provider model catalogs and live upstream model lists.",
    Component: ModelsPage
  },
  {
    id: "pricing",
    label: "Pricing",
    title: "Pricing",
    description: "Provider/model cost overrides used for usage accounting.",
    Component: PricingPage
  },
  {
    id: "usage",
    label: "Usage",
    title: "Usage",
    description: "Request volume, token spend, and recent gateway traffic.",
    Component: UsagePage
  },
  {
    id: "logs",
    label: "Logs",
    title: "Logs",
    description: "Recent gateway request log records and status outcomes.",
    Component: LogsPage
  },
  {
    id: "quota",
    label: "Quota",
    title: "Quota",
    description: "Provider limit usage and reset windows.",
    Component: QuotaPage
  },
  {
    id: "combos",
    label: "Combos",
    title: "Combos",
    description: "Fallback and routing chains for models and accounts.",
    Component: CombosPage
  },
  {
    id: "mcp",
    label: "MCP",
    title: "MCP",
    description: "Tool server health and compact manifest status.",
    Component: McpPage
  },
  {
    id: "settings",
    label: "Settings",
    title: "Settings",
    description: "Runtime defaults and local control-plane configuration.",
    Component: SettingsPage
  },
  {
    id: "diagnostics",
    label: "Diagnostics",
    title: "Diagnostics",
    description: "Release-readiness and control-plane health checks.",
    Component: DiagnosticsPage
  }
] as const;

type PageId = (typeof pages)[number]["id"];

function App() {
  const [activePageId, setActivePageId] = useState<PageId>("dashboard");
  const [apiKeyInput, setApiKeyInput] = useState("");
  const [authRevision, setAuthRevision] = useState(0);
  const activePage = pages.find((page) => page.id === activePageId) ?? pages[0];
  const ActivePageComponent = activePage.Component;
  const hasControlPlaneKey = getControlPlaneKey() !== "";

  function saveKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    saveControlPlaneKey(apiKeyInput);
    setApiKeyInput("");
    setAuthRevision((value) => value + 1);
  }

  function clearKey() {
    clearControlPlaneKey();
    setApiKeyInput("");
    setAuthRevision((value) => value + 1);
  }

  return (
    <div className="min-h-screen bg-zinc-50 text-zinc-950">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r border-zinc-200 bg-white px-5 py-6 lg:block">
        <h1 className="text-xl font-semibold tracking-normal">g0router</h1>
        <p className="mt-1 text-sm text-zinc-500">Control plane</p>

        <nav aria-label="Primary" className="mt-8 space-y-1">
          {pages.map((page) => (
            <button
              key={page.id}
              type="button"
              aria-current={page.id === activePageId ? "page" : undefined}
              onClick={() => setActivePageId(page.id)}
              className="block w-full rounded-md px-3 py-2 text-left text-sm font-medium text-zinc-600 transition hover:bg-zinc-100 hover:text-zinc-950 aria-[current=page]:bg-zinc-950 aria-[current=page]:text-white"
            >
              {page.label}
            </button>
          ))}
        </nav>
      </aside>

      <main className="lg:pl-64">
        <header className="border-b border-zinc-200 bg-white px-5 py-4 lg:px-8">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-medium text-zinc-500">{activePage.label}</p>
              <h2 className="text-2xl font-semibold tracking-normal">{activePage.title}</h2>
              <p className="mt-1 max-w-2xl text-sm leading-6 text-zinc-500">{activePage.description}</p>
            </div>
            <form onSubmit={saveKey} className="flex w-full flex-col gap-2 sm:w-fit sm:flex-row sm:items-end">
              <label className="block text-sm font-medium text-zinc-700">
                Control-plane API key
                <input
                  type="password"
                  value={apiKeyInput}
                  onChange={(event) => setApiKeyInput(event.target.value)}
                  className="mt-1 min-h-10 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-400 sm:w-64"
                  autoComplete="off"
                />
              </label>
              <div className="flex gap-2">
                <button
                  type="submit"
                  className="min-h-10 rounded-md bg-zinc-950 px-4 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
                  disabled={apiKeyInput.trim() === ""}
                >
                  Save key
                </button>
                <button
                  type="button"
                  onClick={clearKey}
                  className="min-h-10 rounded-md border border-zinc-300 px-4 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                  disabled={!hasControlPlaneKey}
                >
                  Clear
                </button>
              </div>
            </form>
          </div>

          <nav aria-label="Sections" className="mt-4 flex gap-2 overflow-x-auto pb-1 lg:hidden">
            {pages.map((page) => (
              <button
                key={page.id}
                type="button"
                aria-current={page.id === activePageId ? "page" : undefined}
                onClick={() => setActivePageId(page.id)}
                className="shrink-0 rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-600 aria-[current=page]:border-zinc-950 aria-[current=page]:bg-zinc-950 aria-[current=page]:text-white"
              >
                {page.label}
              </button>
            ))}
          </nav>
        </header>

        <section className="px-5 py-6 lg:px-8">
          <ActivePageComponent key={`${activePage.id}-${authRevision}`} />
        </section>
      </main>
    </div>
  );
}

export default App;
