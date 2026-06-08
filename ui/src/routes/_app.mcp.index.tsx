import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { MetricCard } from "@/components/common/MetricCard";
import { MetricsGridSkeleton, ErrorState } from "@/components/common/Skeletons";
import type { McpInstance, McpTool, McpClient } from "@/lib/types";

export const Route = createFileRoute("/_app/mcp/")({
  component: McpIndexPage,
});

function McpIndexPage() {
  const clients = useQuery<McpClient[]>({
    queryKey: ["mcp-clients"],
    queryFn: () => apiFetch("/api/mcp/clients"),
  });
  const instances = useQuery<McpInstance[]>({
    queryKey: ["mcp-instances"],
    queryFn: () => apiFetch("/api/mcp/instances"),
  });
  const tools = useQuery<McpTool[]>({
    queryKey: ["mcp-tools"],
    queryFn: () => apiFetch("/api/mcp/tools"),
  });

  const isLoading = clients.isLoading || instances.isLoading || tools.isLoading;
  const isError = clients.isError || instances.isError || tools.isError;
  const error = clients.error || instances.error || tools.error;

  return (
    <div>
      <PageHeader
        title="MCP Overview"
        description="Connected MCP clients, tools and instances."
        icon="hub"
      />
      {isLoading ? (
        <MetricsGridSkeleton count={3} />
      ) : isError ? (
        <ErrorState title="Couldn’t load MCP overview" error={error} />
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <Link to="/mcp/clients">
            <MetricCard
              label="Clients"
              value={clients.data?.length ?? 0}
              icon="devices"
              accent="info"
              hint="Registered MCP clients"
            />
          </Link>
          <Link to="/mcp/instances">
            <MetricCard
              label="Instances"
              value={instances.data?.length ?? 0}
              icon="memory"
              accent="success"
              hint="Running server instances"
            />
          </Link>
          <Link to="/mcp/tools">
            <MetricCard
              label="Tools"
              value={tools.data?.length ?? 0}
              icon="build"
              accent="warning"
              hint="Available tools across all clients"
            />
          </Link>
        </div>
      )}
    </div>
  );
}
