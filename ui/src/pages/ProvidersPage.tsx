import { Panel, StatusPill } from "../components/Primitives";
import type { ProviderConnection } from "../api";

const providers: ProviderConnection[] = [
  { id: "openai-main", provider: "OpenAI", account: "primary", status: "connected", models: 12, lastCheck: "1 min ago" },
  {
    id: "anthropic-ops",
    provider: "Anthropic",
    account: "operations",
    status: "connected",
    models: 8,
    lastCheck: "3 min ago"
  },
  { id: "gemini-lab", provider: "Gemini", account: "lab", status: "degraded", models: 5, lastCheck: "12 min ago" },
  { id: "ollama-local", provider: "Ollama", account: "localhost", status: "disconnected", models: 0, lastCheck: "not checked" }
];

const statusTone = {
  connected: "good",
  degraded: "warn",
  disconnected: "bad"
} as const;

export function ProvidersPage() {
  return (
    <Panel title="Provider connections" description="OAuth and API-token provider accounts available to the proxy.">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {providers.map((provider) => (
          <article key={provider.id} className="rounded-md border border-zinc-200 p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h4 className="font-semibold text-zinc-950">{provider.provider}</h4>
                <p className="mt-1 text-sm text-zinc-500">{provider.account}</p>
              </div>
              <StatusPill tone={statusTone[provider.status]}>{provider.status}</StatusPill>
            </div>
            <dl className="mt-5 grid grid-cols-2 gap-3 text-sm">
              <div>
                <dt className="text-zinc-500">Models</dt>
                <dd className="mt-1 font-semibold text-zinc-950">{provider.models}</dd>
              </div>
              <div>
                <dt className="text-zinc-500">Last check</dt>
                <dd className="mt-1 font-semibold text-zinc-950">{provider.lastCheck}</dd>
              </div>
            </dl>
            <button className="mt-5 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700">
              Configure
            </button>
          </article>
        ))}
      </div>
    </Panel>
  );
}
