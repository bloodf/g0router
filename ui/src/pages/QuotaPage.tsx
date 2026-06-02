import { Panel, ProgressBar, StatusPill } from "../components/Primitives";
import type { QuotaSnapshot } from "../api";

const quotas: QuotaSnapshot[] = [
  { provider: "OpenAI", used: 620000, limit: 1000000, resetAt: "midnight UTC" },
  { provider: "Anthropic", used: 410000, limit: 750000, resetAt: "midnight UTC" },
  { provider: "Gemini", used: 910000, limit: 1000000, resetAt: "rolling 24h" }
];

export function QuotaPage() {
  return (
    <Panel title="Quota monitor" description="Provider usage limits and reset windows for routing decisions.">
      <div className="space-y-5">
        {quotas.map((quota) => {
          const percent = Math.round((quota.used / quota.limit) * 100);

          return (
            <article key={quota.provider} className="rounded-md border border-zinc-200 p-4">
              <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h4 className="font-semibold text-zinc-950">{quota.provider}</h4>
                  <p className="mt-1 text-sm text-zinc-500">
                    {quota.used.toLocaleString()} of {quota.limit.toLocaleString()} tokens
                  </p>
                </div>
                <StatusPill tone={percent > 85 ? "warn" : "good"}>resets {quota.resetAt}</StatusPill>
              </div>
              <ProgressBar label="Quota used" value={percent} />
            </article>
          );
        })}
      </div>
    </Panel>
  );
}
