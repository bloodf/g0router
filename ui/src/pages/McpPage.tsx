import { Panel, StatusPill } from "../components/Primitives";

const instances = [
  {
    name: "atlassian-a",
    launchType: "http",
    tools: 2,
    health: "healthy",
    auth: "connected",
    account: "account-a"
  },
  {
    name: "atlassian-b",
    launchType: "http",
    tools: 2,
    health: "auth required",
    auth: "needs auth",
    account: "account-b"
  },
  {
    name: "expo-local",
    launchType: "npx",
    tools: 5,
    health: "starting",
    auth: "local env",
    account: "workspace"
  }
];

export function McpPage() {
  return (
    <Panel title="MCP gateway" description="Configured MCP instances, accounts, health, and compact tool manifests.">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="text-sm text-zinc-600">{instances.length} instances</div>
        <button className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700">
          Add instance
        </button>
      </div>
      <div className="overflow-hidden rounded-md border border-zinc-200">
        <table className="w-full text-left text-sm">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-3 font-semibold">Instance</th>
              <th className="px-4 py-3 font-semibold">Launch</th>
              <th className="px-4 py-3 font-semibold">Account</th>
              <th className="px-4 py-3 font-semibold">Tools</th>
              <th className="px-4 py-3 font-semibold">Health</th>
              <th className="px-4 py-3 font-semibold">Auth</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200">
            {instances.map((instance) => (
              <tr key={instance.name}>
                <td className="px-4 py-3 font-semibold text-zinc-950">{instance.name}</td>
                <td className="px-4 py-3 text-zinc-600">{instance.launchType}</td>
                <td className="px-4 py-3 text-zinc-600">{instance.account}</td>
                <td className="px-4 py-3 text-zinc-600">{instance.tools}</td>
                <td className="px-4 py-3">
                  <StatusPill tone={instance.health === "healthy" ? "good" : "warn"}>{instance.health}</StatusPill>
                </td>
                <td className="px-4 py-3">
                  {instance.auth === "needs auth" ? (
                    <button className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700">
                      Complete auth
                    </button>
                  ) : (
                    <StatusPill tone="neutral">{instance.auth}</StatusPill>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}
