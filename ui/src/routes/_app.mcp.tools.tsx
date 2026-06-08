import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
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
import { TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import { toast } from "sonner";
import type { McpTool } from "@/lib/types";

export const Route = createFileRoute("/_app/mcp/tools")({
  component: McpToolsPage,
});

function McpToolsPage() {
  const [selected, setSelected] = useState<McpTool | null>(null);
  const [args, setArgs] = useState("{}");
  const [result, setResult] = useState<unknown | null>(null);

  const { data, isLoading, isError, error, refetch } = useQuery<McpTool[]>({
    queryKey: ["mcp-tools"],
    queryFn: () => apiFetch("/api/mcp/tools"),
  });

  const execute = useMutation({
    mutationFn: async ({ name, arguments: parsed }: { name: string; arguments: unknown }) =>
      apiFetch<unknown>(`/api/mcp/tools/${encodeURIComponent(name)}/execute`, {
        method: "POST",
        body: { arguments: parsed, allowed_tools: [name] },
      }),
    onSuccess: (res) => {
      setResult(res);
      toast.success("Tool executed");
    },
    onError: (e: any) => toast.error(e?.message || "Execution failed"),
  });

  const openExecute = (tool: McpTool) => {
    setSelected(tool);
    setArgs("{}");
    setResult(null);
  };

  const run = () => {
    if (!selected) return;
    let parsed: unknown;
    try {
      parsed = JSON.parse(args);
    } catch {
      toast.error("Arguments must be valid JSON");
      return;
    }
    execute.mutate({ name: selected.function.name, arguments: parsed });
  };

  return (
    <div>
      <PageHeader
        title="MCP Tools"
        description="Discover and execute tools exposed by MCP clients and instances."
        icon="build"
      />
      {isLoading ? (
        <TableSkeleton rows={6} columns={4} />
      ) : isError ? (
        <ErrorState title="Couldn’t load tools" error={error} onRetry={refetch} />
      ) : (
        <DataTable
          columns={[
            {
              header: "Name",
              cell: ({ row }) => (
                <code className="font-mono text-xs bg-surface-2 px-1.5 py-0.5 rounded">
                  {row.original.function.name}
                </code>
              ),
            },
            {
              header: "Description",
              cell: ({ row }) => row.original.function.description || "—",
            },
            {
              header: "Parameters",
              cell: ({ row }) =>
                row.original.function.parameters ? (
                  <code className="font-mono text-xs bg-surface-2 px-1.5 py-0.5 rounded">
                    schema
                  </code>
                ) : (
                  "—"
                ),
            },
            {
              id: "actions",
              header: "",
              cell: ({ row }) => (
                <div className="flex justify-end">
                  <Button variant="ghost" size="sm" onClick={() => openExecute(row.original)}>
                    Execute
                  </Button>
                </div>
              ),
            },
          ]}
          data={data ?? []}
          emptyTitle="No tools"
          emptyDescription="Connect an MCP client or instance to see available tools."
          emptyIcon="build"
        />
      )}

      <Dialog
        open={!!selected}
        onOpenChange={(v) => {
          if (!v) {
            setSelected(null);
            setResult(null);
          }
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Execute {selected?.function.name}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <label className="text-xs font-medium block mb-1 text-text-muted">
                Arguments (JSON)
              </label>
              <textarea
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none min-h-[120px] font-mono"
                value={args}
                onChange={(e) => setArgs(e.target.value)}
              />
            </div>
            {result !== null && (
              <div className="space-y-1">
                <label className="text-xs font-medium block text-text-muted">Result</label>
                <pre className="text-xs bg-surface-2 border border-border rounded-lg p-3 overflow-auto max-h-60 font-mono">
                  {JSON.stringify(result, null, 2)}
                </pre>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSelected(null)}>
              Close
            </Button>
            <Button onClick={run} disabled={execute.isPending}>
              {execute.isPending ? "Running…" : "Run"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
