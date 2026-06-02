import { MetricCard, Panel, StatusPill } from "../components/Primitives";

const healthRows = [
  { label: "Gateway status", value: "Ready", tone: "good" as const },
  { label: "Provider health", value: "2 connected", tone: "good" as const },
  { label: "Request flow", value: "Idle", tone: "neutral" as const }
];

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Active providers" value="2" detail="OpenAI and Anthropic accounts loaded" tone="emerald" />
        <MetricCard label="Requests today" value="128" detail="Last request completed 4 minutes ago" tone="sky" />
        <MetricCard label="Fallback events" value="3" detail="All recovered through combo routing" tone="amber" />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <Panel title="Gateway status" description="Operational snapshot for routing, accounts, and request flow.">
          <div className="grid gap-4 sm:grid-cols-3">
            {healthRows.map((row) => (
              <div key={row.label} className="rounded-md border border-zinc-200 p-4">
                <p className="text-sm font-medium text-zinc-500">{row.label}</p>
                <div className="mt-3 flex items-center justify-between gap-3">
                  <p className="text-lg font-semibold text-zinc-950">{row.value}</p>
                  <StatusPill tone={row.tone}>live</StatusPill>
                </div>
              </div>
            ))}
          </div>
        </Panel>

        <Panel title="Operational queue" description="Items waiting for later API wiring.">
          <div className="divide-y divide-zinc-200">
            {["OAuth refresh checks", "Quota collector", "MCP health poll"].map((item) => (
              <div key={item} className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
                <span className="text-sm font-medium text-zinc-700">{item}</span>
                <StatusPill>pending</StatusPill>
              </div>
            ))}
          </div>
        </Panel>
      </div>
    </div>
  );
}
