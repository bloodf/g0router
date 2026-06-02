import { MetricCard, Panel, StatusPill } from "../components/Primitives";
import type { UsageRecord } from "../api";

const requests: UsageRecord[] = [
  {
    id: "req-1092",
    route: "/v1/chat/completions",
    provider: "OpenAI",
    model: "gpt-4.1-mini",
    tokens: 8420,
    costUsd: 0.014,
    latencyMs: 820,
    status: 200
  },
  {
    id: "req-1091",
    route: "/v1/messages",
    provider: "Anthropic",
    model: "claude-sonnet-4",
    tokens: 12104,
    costUsd: 0.092,
    latencyMs: 1280,
    status: 200
  },
  {
    id: "req-1090",
    route: "/v1/chat/completions",
    provider: "Gemini",
    model: "gemini-2.5-flash",
    tokens: 2201,
    costUsd: 0.004,
    latencyMs: 312,
    status: 429
  }
];

const bars = [44, 62, 38, 78, 53, 69, 57];

export function UsagePage() {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Tokens today" value="42.8k" detail="Across 128 completed requests" tone="sky" />
        <MetricCard label="Estimated cost" value="$1.87" detail="Catalog pricing estimate" tone="emerald" />
        <MetricCard label="p95 latency" value="1.28s" detail="Measured at proxy boundary" tone="amber" />
      </div>

      <Panel title="Usage analytics" description="Token, cost, and latency trends for recent gateway traffic.">
        <div className="mb-6 flex h-36 items-end gap-2 border-b border-zinc-200 pb-3">
          {bars.map((value, index) => (
            <div key={`${value}-${index}`} className="flex flex-1 flex-col items-center gap-2">
              <div className="w-full rounded-t bg-zinc-950" style={{ height: `${value}%` }} />
              <span className="text-xs font-medium text-zinc-500">D{index + 1}</span>
            </div>
          ))}
        </div>

        <div className="overflow-hidden rounded-md border border-zinc-200">
          <table className="w-full text-left text-sm">
            <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-4 py-3 font-semibold">Request</th>
                <th className="px-4 py-3 font-semibold">Provider</th>
                <th className="px-4 py-3 font-semibold">Model</th>
                <th className="px-4 py-3 font-semibold">Tokens</th>
                <th className="px-4 py-3 font-semibold">Cost</th>
                <th className="px-4 py-3 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {requests.map((request) => (
                <tr key={request.id}>
                  <td className="px-4 py-3 font-mono text-xs text-zinc-700">{request.route}</td>
                  <td className="px-4 py-3 font-medium text-zinc-950">{request.provider}</td>
                  <td className="px-4 py-3 text-zinc-600">{request.model}</td>
                  <td className="px-4 py-3 text-zinc-600">{request.tokens.toLocaleString()}</td>
                  <td className="px-4 py-3 text-zinc-600">${request.costUsd.toFixed(3)}</td>
                  <td className="px-4 py-3">
                    <StatusPill tone={request.status === 200 ? "good" : "warn"}>{request.status}</StatusPill>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>
    </div>
  );
}
