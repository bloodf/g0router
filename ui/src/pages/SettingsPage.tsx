import { Panel, StatusPill } from "../components/Primitives";

const settings = [
  { label: "Default upstream timeout", value: "30s" },
  { label: "Streaming flush interval", value: "100ms" },
  { label: "Usage logging", value: "enabled" },
  { label: "Cost catalog", value: "bundled" }
];

export function SettingsPage() {
  return (
    <Panel title="Runtime settings" description="Gateway defaults that affect proxy behavior and local control-plane access.">
      <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
        <div className="space-y-3">
          {settings.map((setting) => (
            <div key={setting.label} className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 px-4 py-3">
              <span className="text-sm font-medium text-zinc-700">{setting.label}</span>
              <span className="text-sm font-semibold text-zinc-950">{setting.value}</span>
            </div>
          ))}
        </div>

        <div className="rounded-md border border-zinc-200 p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <h4 className="font-semibold text-zinc-950">Control plane access</h4>
              <p className="mt-2 text-sm leading-6 text-zinc-500">
                Localhost-only UI until the embed and management API tasks expose deployment controls.
              </p>
            </div>
            <StatusPill tone="good">local</StatusPill>
          </div>
          <button className="mt-5 rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700">
            Save settings
          </button>
        </div>
      </div>
    </Panel>
  );
}
