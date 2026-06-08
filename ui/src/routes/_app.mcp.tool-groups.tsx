import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import type { McpToolGroup } from "@/lib/types";

export const Route = createFileRoute("/_app/mcp/tool-groups")({
  component: () => (
    <CrudPage<McpToolGroup>
      title="MCP Tool Groups"
      description="Group and allow-list MCP tools for routing."
      icon="workspaces"
      endpoint="/api/mcp/tool-groups"
      queryKey={["mcp-tool-groups"]}
      emptyTitle="No tool groups"
      emptyDescription="Create groups to control which MCP tools are exposed to requests."
      fields={[
        { name: "name", label: "Name", required: true },
        { name: "tool_ids", label: "Tool IDs (comma-separated)", type: "textarea" },
        { name: "is_active", label: "Active", type: "switch" },
      ]}
      initialValues={(row) => ({
        name: row?.name ?? "",
        tool_ids: row?.tool_ids?.join(", ") ?? "",
        is_active: row?.is_active ?? true,
      })}
      transformBody={(values) => ({
        ...values,
        tool_ids: values.tool_ids
          ? String(values.tool_ids)
              .split(",")
              .map((s: string) => s.trim())
              .filter(Boolean)
          : [],
      })}
      columns={[
        { header: "Name", accessorKey: "name" },
        {
          header: "Tools",
          cell: ({ row }) => row.original.tool_ids?.length ?? 0,
        },
        {
          header: "Status",
          cell: ({ row }) => (
            <StatusBadge variant={row.original.is_active ? "success" : "muted"} dot>
              {row.original.is_active ? "active" : "inactive"}
            </StatusBadge>
          ),
        },
      ]}
    />
  ),
});
