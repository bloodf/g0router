import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PageHeader } from "@/components/common/PageHeader";
import { DataTable } from "@/components/common/DataTable";
import { StatusBadge } from "@/components/common/StatusBadge";
import { TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import { toast } from "sonner";
import type { McpInstance } from "@/lib/types";

export const Route = createFileRoute("/_app/mcp/instances")({
  component: McpInstancesPage,
});

function McpInstancesPage() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [values, setValues] = useState<Record<string, any>>({
    name: "",
    server_key: "",
    launch_type: "command",
    transport: "stdio",
    command: "",
    args: "",
    url: "",
    env: "",
    is_active: true,
  });

  const { data, isLoading, isError, error, refetch } = useQuery<McpInstance[]>({
    queryKey: ["mcp-instances"],
    queryFn: () => apiFetch("/api/mcp/instances"),
  });

  const create = useMutation({
    mutationFn: (body: any) =>
      apiFetch<McpInstance>("/api/mcp/instances", { method: "POST", body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["mcp-instances"] });
      toast.success("Instance created");
      setOpen(false);
      setValues({
        name: "",
        server_key: "",
        launch_type: "command",
        transport: "stdio",
        command: "",
        args: "",
        url: "",
        env: "",
        is_active: true,
      });
    },
    onError: (e: any) => toast.error(e?.message || "Failed to create instance"),
  });

  const del = useMutation({
    mutationFn: (id: string) => apiFetch(`/api/mcp/instances/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["mcp-instances"] });
      toast.success("Instance deleted");
    },
    onError: (e: any) => toast.error(e?.message || "Failed to delete instance"),
  });

  const parseEnv = (raw: string) => {
    const out: Record<string, string> = {};
    raw.split("\n").forEach((line) => {
      const [k, ...rest] = line.split("=");
      if (k && rest.length) out[k.trim()] = rest.join("=").trim();
    });
    return out;
  };

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const body: any = {
      name: values.name,
      server_key: values.server_key,
      launch_type: values.launch_type,
      transport: values.transport,
      is_active: values.is_active,
    };
    if (values.command) body.command = values.command;
    if (values.args)
      body.args = String(values.args)
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
    if (values.url) body.url = values.url;
    if (values.env) body.env = parseEnv(values.env);
    create.mutate(body);
  };

  return (
    <div>
      <PageHeader
        title="MCP Instances"
        description="Local MCP server processes and remote endpoints."
        icon="memory"
        actions={<Button onClick={() => setOpen(true)}>Create instance</Button>}
      />
      {isLoading ? (
        <TableSkeleton rows={5} columns={5} />
      ) : isError ? (
        <ErrorState title="Couldn’t load instances" error={error} onRetry={refetch} />
      ) : (
        <DataTable
          columns={[
            { header: "Name", accessorKey: "Name" },
            { header: "Transport", accessorKey: "Transport" },
            {
              header: "Command / URL",
              cell: ({ row }) => row.original.URL || row.original.Command || "—",
            },
            {
              header: "Health",
              cell: ({ row }) => (
                <StatusBadge
                  variant={
                    row.original.HealthStatus === "healthy"
                      ? "success"
                      : row.original.HealthStatus === "unhealthy"
                        ? "danger"
                        : "muted"
                  }
                  dot
                >
                  {row.original.HealthStatus || "unknown"}
                </StatusBadge>
              ),
            },
            {
              header: "Tools",
              cell: ({ row }) => row.original.ToolManifest?.Tools?.length ?? 0,
            },
            {
              id: "actions",
              header: "",
              cell: ({ row }) => (
                <div className="flex items-center gap-1 justify-end">
                  <Button variant="ghost" size="sm" onClick={() => del.mutate(row.original.ID)}>
                    Delete
                  </Button>
                </div>
              ),
            },
          ]}
          data={data ?? []}
          emptyTitle="No instances"
          emptyDescription="Create an MCP instance to expose its tools."
          emptyIcon="memory"
        />
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Create MCP instance</DialogTitle>
          </DialogHeader>
          <form onSubmit={submit} className="space-y-3">
            <div>
              <label htmlFor="mcp-name" className="text-xs font-medium block mb-1 text-text-muted">Name *</label>
              <input
                id="mcp-name"
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                value={values.name}
                onChange={(e) => setValues((v) => ({ ...v, name: e.target.value }))}
                required
              />
            </div>
            <div>
              <label htmlFor="mcp-server-key" className="text-xs font-medium block mb-1 text-text-muted">Server key *</label>
              <input
                id="mcp-server-key"
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                value={values.server_key}
                onChange={(e) => setValues((v) => ({ ...v, server_key: e.target.value }))}
                required
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label htmlFor="mcp-launch-type" className="text-xs font-medium block mb-1 text-text-muted">
                  Launch type *
                </label>
                <select
                  id="mcp-launch-type"
                  className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                  value={values.launch_type}
                  onChange={(e) => setValues((v) => ({ ...v, launch_type: e.target.value }))}
                >
                  <option value="command">command</option>
                  <option value="npx">npx</option>
                  <option value="docker">docker</option>
                  <option value="http">http</option>
                </select>
              </div>
              <div>
                <label htmlFor="mcp-transport" className="text-xs font-medium block mb-1 text-text-muted">
                  Transport *
                </label>
                <select
                  id="mcp-transport"
                  className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                  value={values.transport}
                  onChange={(e) => setValues((v) => ({ ...v, transport: e.target.value }))}
                >
                  <option value="stdio">stdio</option>
                  <option value="sse">sse</option>
                  <option value="streamable-http">streamable-http</option>
                </select>
              </div>
            </div>
            <div>
              <label htmlFor="mcp-command" className="text-xs font-medium block mb-1 text-text-muted">Command</label>
              <input
                id="mcp-command"
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                value={values.command}
                onChange={(e) => setValues((v) => ({ ...v, command: e.target.value }))}
              />
            </div>
            <div>
              <label htmlFor="mcp-args" className="text-xs font-medium block mb-1 text-text-muted">
                Args (comma-separated)
              </label>
              <input
                id="mcp-args"
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                value={values.args}
                onChange={(e) => setValues((v) => ({ ...v, args: e.target.value }))}
              />
            </div>
            <div>
              <label htmlFor="mcp-url" className="text-xs font-medium block mb-1 text-text-muted">URL</label>
              <input
                id="mcp-url"
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                value={values.url}
                onChange={(e) => setValues((v) => ({ ...v, url: e.target.value }))}
              />
            </div>
            <div>
              <label htmlFor="mcp-env" className="text-xs font-medium block mb-1 text-text-muted">
                Env (KEY=value per line)
              </label>
              <textarea
                id="mcp-env"
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none min-h-[80px]"
                value={values.env}
                onChange={(e) => setValues((v) => ({ ...v, env: e.target.value }))}
              />
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                className="accent-brand-500 w-4 h-4"
                checked={values.is_active}
                onChange={(e) => setValues((v) => ({ ...v, is_active: e.target.checked }))}
              />
              <span className="text-sm">Active</span>
            </label>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={create.isPending}>
                {create.isPending ? "Creating…" : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
