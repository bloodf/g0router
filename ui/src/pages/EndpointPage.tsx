import { Panel, StatusPill } from "../components/Primitives";

const apiKeys = [
  { name: "local-admin", prefix: "g0_live_3fb2", scope: "control plane", status: "enabled" },
  { name: "proxy-readonly", prefix: "g0_live_a917", scope: "usage read", status: "enabled" }
];

const filters = ["RTK autodetect", "Caveman compression", "Streaming accumulator"];

export function EndpointPage() {
  return (
    <Panel title="Endpoint controls" description="API key, request transformation, and endpoint protection controls.">
      <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <div>
          <div className="mb-3 flex items-center justify-between gap-3">
            <h4 className="text-sm font-semibold text-zinc-700">API keys</h4>
            <button className="rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white">Create key</button>
          </div>
          <div className="overflow-hidden rounded-md border border-zinc-200">
            <table className="w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                <tr>
                  <th className="px-4 py-3 font-semibold">Name</th>
                  <th className="px-4 py-3 font-semibold">Prefix</th>
                  <th className="px-4 py-3 font-semibold">Scope</th>
                  <th className="px-4 py-3 font-semibold">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {apiKeys.map((key) => (
                  <tr key={key.name}>
                    <td className="px-4 py-3 font-medium text-zinc-950">{key.name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-zinc-600">{key.prefix}</td>
                    <td className="px-4 py-3 text-zinc-600">{key.scope}</td>
                    <td className="px-4 py-3">
                      <StatusPill tone="good">{key.status}</StatusPill>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-zinc-700">Request controls</h4>
          {filters.map((filter) => (
            <label key={filter} className="flex items-center justify-between rounded-md border border-zinc-200 px-4 py-3">
              <span className="text-sm font-medium text-zinc-700">{filter}</span>
              <input type="checkbox" defaultChecked className="h-4 w-4 accent-zinc-950" />
            </label>
          ))}
        </div>
      </div>
    </Panel>
  );
}
