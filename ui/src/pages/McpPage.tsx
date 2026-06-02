import { Panel, StatusPill } from "../components/Primitives";

const servers = [
  { name: "filesystem", tools: 12, status: "available", manifest: "compact" },
  { name: "github", tools: 9, status: "available", manifest: "compact" },
  { name: "local-shell", tools: 4, status: "paused", manifest: "pending" }
];

export function McpPage() {
  return (
    <Panel title="MCP gateway" description="Connected MCP servers, compact manifests, and tool availability.">
      <div className="overflow-hidden rounded-md border border-zinc-200">
        <table className="w-full text-left text-sm">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-3 font-semibold">Server</th>
              <th className="px-4 py-3 font-semibold">Tools</th>
              <th className="px-4 py-3 font-semibold">Manifest</th>
              <th className="px-4 py-3 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200">
            {servers.map((server) => (
              <tr key={server.name}>
                <td className="px-4 py-3 font-semibold text-zinc-950">{server.name}</td>
                <td className="px-4 py-3 text-zinc-600">{server.tools}</td>
                <td className="px-4 py-3 text-zinc-600">{server.manifest}</td>
                <td className="px-4 py-3">
                  <StatusPill tone={server.status === "available" ? "good" : "warn"}>{server.status}</StatusPill>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}
