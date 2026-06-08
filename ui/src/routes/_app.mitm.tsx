import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch, ApiError } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Icon } from "@/components/common/Icon";
import { CardSkeleton, ErrorState } from "@/components/common/Skeletons";
import { toast } from "sonner";

interface MITMTool {
  name: string;
  proxy_env: string;
  hosts_line: string;
  enabled?: boolean;
}

interface MITMStatus {
  running: boolean;
  addr: string;
  tools: MITMTool[];
}

export const Route = createFileRoute("/_app/mitm")({
  component: MITMPage,
});

function MITMPage() {
  const qc = useQueryClient();
  const { data, isLoading, isError, error, refetch } = useQuery<MITMStatus>({
    queryKey: ["mitm-status"],
    queryFn: () => apiFetch("/api/mitm/status"),
    retry: false,
  });

  const toggle = useMutation({
    mutationFn: (enabled: boolean) =>
      apiFetch("/api/mitm/toggle", {
        method: "POST",
        body: { enabled },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["mitm-status"] });
      toast.success("MITM proxy updated");
    },
    onError: (e: any) => toast.error(e?.message || "Failed to toggle MITM proxy"),
  });

  const toggleTool = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      apiFetch(`/api/mitm/tools/${encodeURIComponent(id)}`, {
        method: "POST",
        body: { enabled },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["mitm-status"] });
      toast.success("Tool updated");
    },
    onError: (e: any) => toast.error(e?.message || "Failed to update tool"),
  });

  if (isError && error instanceof ApiError && error.status === 404) {
    return (
      <div>
        <PageHeader
          title="MITM"
          description="Intercept AI tool HTTPS traffic and route it through the gateway."
          icon="security"
        />
        <Card className="card-elev border-border p-8">
          <div className="flex flex-col items-center text-center gap-3">
            <div className="w-12 h-12 rounded-xl bg-surface-2 flex items-center justify-center">
              <Icon name="block" size={28} className="text-text-muted" />
            </div>
            <h3 className="font-semibold">MITM proxy is disabled</h3>
            <p className="text-sm text-text-muted max-w-md">
              Enable the{" "}
              <code className="text-xs bg-surface-2 px-1.5 py-0.5 rounded">
                mitm_proxy
              </code>{" "}
              feature flag to use this feature.
            </p>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="MITM"
        description="Intercept AI tool HTTPS traffic and route it through the gateway."
        icon="security"
      />
      {isLoading ? (
        <div className="space-y-4">
          <CardSkeleton lines={3} />
          <CardSkeleton lines={4} />
        </div>
      ) : isError ? (
        <ErrorState
          title="Couldn’t load MITM status"
          error={error}
          onRetry={() => refetch()}
        />
      ) : (
        <div className="space-y-4">
          <Card className="card-elev border-border p-5 space-y-4">
            <div className="flex items-start justify-between gap-4 flex-wrap">
              <div>
                <h3 className="font-semibold flex items-center gap-2">
                  <Icon
                    name={data?.running ? "check_circle" : "cancel"}
                    size={20}
                    className={data?.running ? "text-success" : "text-text-muted"}
                  />
                  Status: {data?.running ? "Running" : "Stopped"}
                </h3>
                <p className="text-sm text-text-muted mt-1">
                  Address: {data?.addr || "not started"}
                </p>
              </div>
              <Button
                variant={data?.running ? "outline" : "default"}
                onClick={() => toggle.mutate(!data?.running)}
                disabled={toggle.isPending}
              >
                <Icon
                  name={data?.running ? "stop" : "play_arrow"}
                  size={16}
                  className="mr-1.5"
                />
                {data?.running ? "Stop" : "Start"}
              </Button>
            </div>
            <div className="flex gap-2">
              <Button asChild variant="outline">
                <a href="/api/mitm/ca-cert" download="g0router-mitm-ca.crt">
                  <Icon name="download" size={16} className="mr-1.5" />
                  Download CA Cert
                </a>
              </Button>
            </div>
          </Card>

          <Card className="card-elev border-border p-5">
            <h3 className="font-semibold mb-3">Tool Interception</h3>
            {data?.tools.length === 0 ? (
              <p className="text-sm text-text-muted">No tools configured.</p>
            ) : (
              <div className="divide-y divide-border">
                {data?.tools.map((tool) => (
                  <div
                    key={tool.name}
                    className="py-3 flex items-center justify-between gap-4"
                  >
                    <div className="min-w-0">
                      <p className="font-medium">{tool.name}</p>
                      <p className="text-xs text-text-muted font-mono truncate">
                        {tool.proxy_env}
                      </p>
                    </div>
                    <Switch
                      checked={!!tool.enabled}
                      onCheckedChange={(checked) =>
                        toggleTool.mutate({ id: tool.name, enabled: checked })
                      }
                      disabled={toggleTool.isPending}
                    />
                  </div>
                ))}
              </div>
            )}
          </Card>
        </div>
      )}
    </div>
  );
}
