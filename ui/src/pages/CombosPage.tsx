import { Panel, StatusPill } from "../components/Primitives";
import type { ComboRoute } from "../api";

const combos: ComboRoute[] = [
  { name: "balanced-chat", strategy: "fallback", providers: ["OpenAI", "Anthropic", "Gemini"] },
  { name: "low-latency", strategy: "round robin", providers: ["Gemini", "Ollama"] },
  { name: "research-heavy", strategy: "ordered", providers: ["Anthropic", "OpenAI"] }
];

export function CombosPage() {
  return (
    <Panel title="Combo routing" description="Reusable routing chains for fallback, round-robin, and account selection.">
      <div className="space-y-3">
        {combos.map((combo) => (
          <article key={combo.name} className="rounded-md border border-zinc-200 p-4">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h4 className="font-semibold text-zinc-950">{combo.name}</h4>
                <p className="mt-1 text-sm text-zinc-500">{combo.providers.join(" -> ")}</p>
              </div>
              <StatusPill>{combo.strategy}</StatusPill>
            </div>
          </article>
        ))}
      </div>
    </Panel>
  );
}
