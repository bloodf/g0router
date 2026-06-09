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
  const {
    data: clients = [],
    isLoading: clientsLoading,
    isError: clientsError,
    error: clientsErr,
  } = useQuery<McpClient[]>({
    queryKey: ["mcp-clients"],
    queryFn: () => apiFetch("/api/mcp/clients"),
  });
  const {
    data: instances = [],
    isLoading: instancesLoading,
    isError: instancesError,
    error: instancesErr,
  } = useQuery<McpInstance[]>({
    queryKey: ["mcp-instances"],
    queryFn: () => apiFetch("/api/mcp/instances"),
  });
  const {
    data: tools = [],
    isLoading: toolsLoading,
    isError: toolsError,
    error: toolsErr,
  } = useQuery<McpTool[]>({
    queryKey: ["mcp-tools"],
    queryFn: () => apiFetch("/api/mcp/tools"),
  });

  const isLoading = clientsLoading || instancesLoading || toolsLoading;
  const isError = clientsError || instancesError || toolsError;
  const error = clientsErr || instancesErr || toolsErr;

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
              value={clients.length}
              icon="devices"
              accent="info"
              hint="Registered MCP clients"
            />
          </Link>
          <Link to="/mcp/instances">
            <MetricCard
              label="Instances"
              value={instances.length}
              icon="memory"
              accent="success"
              hint="Running server instances"
            />
          </Link>
          <Link to="/mcp/tools">
            <MetricCard
              label="Tools"
              value={tools.length}
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
