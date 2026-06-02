import { useState } from "react";
import "./index.css";
import { CombosPage } from "./pages/CombosPage";
import { DashboardPage } from "./pages/DashboardPage";
import { EndpointPage } from "./pages/EndpointPage";
import { McpPage } from "./pages/McpPage";
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
    component: <DashboardPage />
  },
  {
    id: "endpoint",
    label: "Endpoint",
    title: "Endpoint configuration",
    description: "API keys, RTK, caveman, and request controls.",
    component: <EndpointPage />
  },
  {
    id: "providers",
    label: "Providers",
    title: "Providers",
    description: "Connection status and provider account management.",
    component: <ProvidersPage />
  },
  {
    id: "usage",
    label: "Usage",
    title: "Usage",
    description: "Request volume, token spend, and recent gateway traffic.",
    component: <UsagePage />
  },
  {
    id: "quota",
    label: "Quota",
    title: "Quota",
    description: "Provider limit usage and reset windows.",
    component: <QuotaPage />
  },
  {
    id: "combos",
    label: "Combos",
    title: "Combos",
    description: "Fallback and routing chains for models and accounts.",
    component: <CombosPage />
  },
  {
    id: "mcp",
    label: "MCP",
    title: "MCP",
    description: "Tool server health and compact manifest status.",
    component: <McpPage />
  },
  {
    id: "settings",
    label: "Settings",
    title: "Settings",
    description: "Runtime defaults and local control-plane configuration.",
    component: <SettingsPage />
  }
] as const;

type PageId = (typeof pages)[number]["id"];

function App() {
  const [activePageId, setActivePageId] = useState<PageId>("dashboard");
  const activePage = pages.find((page) => page.id === activePageId) ?? pages[0];

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
            <div className="inline-flex w-fit items-center gap-2 rounded-md border border-zinc-200 px-3 py-2 text-sm font-medium text-zinc-700">
              <span className="h-2 w-2 rounded-full bg-emerald-500" />
              Local control plane
            </div>
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
          {activePage.component}
          {activePageId === "dashboard" ? (
            <div className="mt-6 grid gap-4 xl:grid-cols-2">
              <EndpointPage />
              <ProvidersPage />
              <UsagePage />
              <QuotaPage />
              <CombosPage />
              <McpPage />
              <SettingsPage />
            </div>
          ) : null}
        </section>
      </main>
    </div>
  );
}

export default App;
