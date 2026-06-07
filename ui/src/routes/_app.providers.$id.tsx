import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/PageHeader";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Icon } from "@/components/common/Icon";
import { ProviderTopology } from "@/components/topology/ProviderTopology";
import { DetailHeaderSkeleton, TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import type { Connection, Model, Provider } from "@/lib/mocks/types";
import { toast } from "sonner";

export const Route = createFileRoute("/_app/providers/$id")({
  component: ProviderDetail,
});

function ProviderDetail() {
  const { id } = Route.useParams();
  const {
    data: provider,
    isLoading: providerLoading,
    isError: providerError,
    error: providerErr,
    refetch: refetchProvider,
  } = useQuery<Provider>({
    queryKey: ["provider", id],
    queryFn: () => apiFetch(`/api/providers/${id}`),
  });
  const { data: conns = [], refetch: refetchConn } = useQuery<Connection[]>({
    queryKey: ["provider", id, "connections"],
    queryFn: () => apiFetch(`/api/providers/${id}/connections`),
  });
  const modelsQ = useQuery<Model[]>({
    queryKey: ["provider", id, "models"],
    queryFn: () => apiFetch(`/api/providers/${id}/models`),
  });
  const models = modelsQ.data ?? [];

  const testConn = async (cid: string) => {
    const r = await apiFetch(`/api/connections/${cid}/test`, { method: "POST" });
    toast[r.ok ? "success" : "error"](
      r.ok ? `OK (${r.latency_ms}ms)` : "Failed",
    );
  };

  if (providerError)
    return (
      <ErrorState
        title="Couldn’t load provider"
        error={providerErr}
        onRetry={() => refetchProvider()}
      />
    );

  if (providerLoading || !provider)
    return (
      <div>
        <DetailHeaderSkeleton />
        <TableSkeleton rows={6} columns={5} />
      </div>
    );

  return (
    <div>
      <div className="flex items-center gap-2 text-xs text-text-muted mb-3">
        <Link to="/providers" className="hover:text-foreground">
          Providers
        </Link>
        <Icon name="chevron_right" size={14} />
        <span>{provider.display_name}</span>
      </div>

      <div className="flex items-start justify-between mb-6 gap-4">
        <div className="flex items-start gap-4">
          <ProviderIcon provider={provider.id} size={56} />
          <div>
            <h1 className="text-2xl font-semibold">{provider.display_name}</h1>
            <p className="text-sm text-text-muted mt-1">{provider.description}</p>
            <div className="flex flex-wrap gap-1 mt-2">
              {provider.auth_types.map((a) => (
                <StatusBadge key={a} variant="primary">
                  {a.replace("_", " ")}
                </StatusBadge>
              ))}
              {provider.capabilities.map((c) => (
                <StatusBadge key={c} variant="muted">
                  {c}
                </StatusBadge>
              ))}
            </div>
          </div>
        </div>
        <Button className="btn-cta">
          <Icon name="add" size={16} className="mr-1.5" />
          New connection
        </Button>
      </div>

      <Tabs defaultValue="connections">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="connections">Connections</TabsTrigger>
          <TabsTrigger value="models">Models</TabsTrigger>
          <TabsTrigger value="topology">Topology</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-4">
          <Card className="p-6 card-elev border-border">
            <h3 className="font-semibold mb-2">About</h3>
            <p className="text-sm text-text-muted">{provider.description}</p>
            <div className="grid grid-cols-3 gap-4 mt-4 pt-4 border-t border-border">
              <div>
                <div className="text-xs text-text-muted">Connections</div>
                <div className="text-xl font-semibold">{provider.connection_count}</div>
              </div>
              <div>
                <div className="text-xs text-text-muted">Models</div>
                <div className="text-xl font-semibold">{models.length}</div>
              </div>
              <div>
                <div className="text-xs text-text-muted">Status</div>
                <StatusBadge
                  variant={
                    provider.status === "active"
                      ? "success"
                      : provider.status === "needs_reauth"
                        ? "warning"
                        : "danger"
                  }
                  dot
                >
                  {provider.status}
                </StatusBadge>
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="connections" className="mt-4">
          <Card className="border-border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-surface-2 text-[11px] uppercase tracking-wider text-text-muted text-left">
                <tr>
                  <th className="px-4 py-2">Name</th>
                  <th className="px-4 py-2">Auth</th>
                  <th className="px-4 py-2">Models</th>
                  <th className="px-4 py-2">Status</th>
                  <th className="px-4 py-2 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {conns.map((c) => (
                  <tr key={c.id} className="border-t border-border">
                    <td className="px-4 py-2 font-medium">{c.name}</td>
                    <td className="px-4 py-2">{c.auth_type}</td>
                    <td className="px-4 py-2 text-xs text-text-muted">{c.models.length}</td>
                    <td className="px-4 py-2">
                      <StatusBadge
                        variant={c.is_active ? "success" : "muted"}
                        dot
                      >
                        {c.is_active ? "active" : "inactive"}
                      </StatusBadge>
                    </td>
                    <td className="px-4 py-2 text-right">
                      <Button variant="ghost" size="sm" onClick={() => testConn(c.id)}>
                        <Icon name="play_circle" size={14} />
                      </Button>
                    </td>
                  </tr>
                ))}
                {!conns.length && (
                  <tr>
                    <td colSpan={5} className="py-8 text-center text-sm text-text-muted">
                      No connections yet
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </Card>
        </TabsContent>

        <TabsContent value="models" className="mt-4">
          <Card className="border-border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-surface-2 text-[11px] uppercase tracking-wider text-text-muted text-left">
                <tr>
                  <th className="px-4 py-2">Model</th>
                  <th className="px-4 py-2 text-right">Input ($/M)</th>
                  <th className="px-4 py-2 text-right">Output ($/M)</th>
                  <th className="px-4 py-2 text-right">Context</th>
                  <th className="px-4 py-2">Status</th>
                </tr>
              </thead>
              <tbody>
                {models.map((m) => (
                  <tr key={m.id} className="border-t border-border">
                    <td className="px-4 py-2 font-mono text-xs">{m.name}</td>
                    <td className="px-4 py-2 text-right tabular-nums">${m.input_cost}</td>
                    <td className="px-4 py-2 text-right tabular-nums">${m.output_cost}</td>
                    <td className="px-4 py-2 text-right tabular-nums text-xs text-text-muted">
                      {Intl.NumberFormat("en", { notation: "compact" }).format(m.context_window)}
                    </td>
                    <td className="px-4 py-2">
                      <StatusBadge variant={m.is_disabled ? "muted" : "success"} dot>
                        {m.is_disabled ? "disabled" : "enabled"}
                      </StatusBadge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        </TabsContent>

        <TabsContent value="topology" className="mt-4">
          <ProviderTopology providerFilter={id} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
