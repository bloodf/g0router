import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Toggle } from "@/components/ui/toggle";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { McpToolGroupModal } from "@/components/mcp/mcp-tool-group-modal";
import { useNotificationStore } from "@/stores/notification";
import type { McpTool, McpToolGroup } from "@/lib/types";

export const Route = createFileRoute("/mcp/tools")({
  component: McpToolsPage,
});

// McpToolsPage (PAR-UI-130 /mcp/tools, g0router-EXTRA) lists MCP tools from
// GET /api/mcp/tools (each with an Execute action → POST …/{name}/execute) and
// tool-groups from GET /api/mcp/tool-groups with create/edit (McpToolGroupModal,
// POST/PUT), is_active Toggle, and delete (ConfirmModal). Tool-groups use
// snake_case keys (§1.2/§1.4). Variant-HAVE; no Go backend yet (§8 ESC-1b).
function McpToolsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [tools, setTools] = React.useState<McpTool[]>([]);
  const [groups, setGroups] = React.useState<McpToolGroup[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [results, setResults] = React.useState<Record<string, string>>({});
  const [executing, setExecuting] = React.useState<string | null>(null);
  const [editing, setEditing] = React.useState<McpToolGroup | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<McpToolGroup | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    Promise.all([
      apiFetch<McpTool[]>("/api/mcp/tools"),
      apiFetch<McpToolGroup[]>("/api/mcp/tool-groups"),
    ])
      .then(([t, g]) => {
        setTools(t ?? []);
        setGroups(g ?? []);
        setLoading(false);
      })
      .catch(() => {
        setTools([]);
        setGroups([]);
        setLoading(false);
        pushToast({ message: "Failed to load MCP tools" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function execute(tool: McpTool) {
    const name = tool.function.name;
    setExecuting(name);
    try {
      const res = await apiFetch<{ result: string }>(
        `/api/mcp/tools/${name}/execute`,
        { method: "POST", body: JSON.stringify({ arguments: {} }) }
      );
      setResults((prev) => ({ ...prev, [name]: res?.result ?? "" }));
    } catch {
      pushToast({ message: "Failed to execute the tool" });
    } finally {
      setExecuting(null);
    }
  }

  async function setActive(group: McpToolGroup, active: boolean) {
    setGroups((prev) =>
      prev.map((g) => (g.id === group.id ? { ...g, is_active: active } : g))
    );
    try {
      await apiFetch(`/api/mcp/tool-groups/${group.id}`, {
        method: "PUT",
        body: JSON.stringify({ ...group, is_active: active }),
      });
    } catch {
      pushToast({ message: "Failed to update the tool group" });
      load();
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/mcp/tool-groups/${deleting.id}`, { method: "DELETE" });
      setGroups((prev) => prev.filter((g) => g.id !== deleting.id));
      pushToast({ message: "Tool group deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the tool group" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">MCP Tools</h1>
        <Button
          data-testid="mcp-tool-group-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New tool group
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : (
        <>
          <section className="flex flex-col gap-2">
            <h2 className="text-sm font-semibold text-foreground">Tools</h2>
            {tools.length === 0 ? (
              <p className="text-sm text-muted-foreground">No MCP tools yet.</p>
            ) : (
              tools.map((tool) => (
                <div
                  key={tool.function.name}
                  data-testid="mcp-tool-row"
                  className="flex flex-col gap-2 rounded-lg border border-border px-4 py-3"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-foreground">
                        {tool.function.name}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {tool.function.description}
                      </p>
                    </div>
                    <Button
                      data-testid="mcp-tool-execute"
                      variant="ghost"
                      size="sm"
                      loading={executing === tool.function.name}
                      onClick={() => execute(tool)}
                    >
                      Execute
                    </Button>
                  </div>
                  {results[tool.function.name] ? (
                    <pre
                      data-testid="mcp-tool-result"
                      className="rounded bg-muted px-3 py-2 text-xs text-foreground"
                    >
                      {results[tool.function.name]}
                    </pre>
                  ) : null}
                </div>
              ))
            )}
          </section>

          <section className="flex flex-col gap-2">
            <h2 className="text-sm font-semibold text-foreground">Tool groups</h2>
            {groups.length === 0 ? (
              <p className="text-sm text-muted-foreground">No tool groups yet.</p>
            ) : (
              groups.map((group) => (
                <div
                  key={group.id}
                  data-testid="mcp-tool-group-row"
                  className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
                >
                  <div className="flex items-center gap-3">
                    <div>
                      <p className="text-sm font-medium text-foreground">
                        {group.name}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {group.tool_ids.join(", ")}
                      </p>
                    </div>
                    <Badge variant="neutral" size="sm">
                      {group.tool_ids.length} tools
                    </Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    <Toggle
                      checked={group.is_active}
                      onCheckedChange={(checked) => setActive(group, checked)}
                      aria-label={`Toggle ${group.name}`}
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setEditing(group)}
                    >
                      Edit
                    </Button>
                    <Button
                      data-testid="mcp-tool-group-delete"
                      variant="danger"
                      size="sm"
                      onClick={() => setDeleting(group)}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              ))
            )}
          </section>
        </>
      )}

      <McpToolGroupModal
        open={creating || editing !== null}
        group={editing}
        tools={tools}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete tool group"
        message={`Delete "${deleting?.name ?? ""}"? This cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        loading={deleteBusy}
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </div>
  );
}
