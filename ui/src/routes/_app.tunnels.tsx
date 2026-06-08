import { createFileRoute, Link, useRouterState } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { CardsGridSkeleton, ErrorState } from "@/components/common/Skeletons";
import { CopyButton } from "@/components/common/CopyButton";
import { Icon } from "@/components/common/Icon";
import { StatusBadge } from "@/components/common/StatusBadge";
import type { Tunnel } from "@/lib/types";
import { toast } from "sonner";

export const Route = createFileRoute("/_app/tunnels")({
  component: TunnelsPage,
});

interface TunnelMeta {
  type: "cloudflare" | "tailscale";
  title: string;
  icon: string;
  color: string;
  description: string;
  docsHref: string;
  details: string[];
}

const META: TunnelMeta[] = [
  {
    type: "cloudflare",
    title: "Cloudflare Tunnel",
    icon: "cloud",
    color: "text-info",
    description:
      "Expose the endpoint over a trycloudflare.com URL. Zero config, no port forwarding, encrypted via Cloudflare's global network.",
    docsHref:
      "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/",
    details: [
      "Public URL anyone can call with a valid API key",
      "Auto-renewing TLS, no DNS setup required",
      "Best for sharing with teammates over the internet",
    ],
  },
  {
    type: "tailscale",
    title: "Tailscale",
    icon: "hub",
    color: "text-brand-600",
    description:
      "Reachable only from devices on your tailnet. Best for teams that want a private endpoint accessible without a public URL.",
    docsHref: "https://tailscale.com/kb/1080/cli",
    details: [
      "MagicDNS hostname inside your tailnet",
      "Zero-trust by default — no public IP",
      "Ideal for internal services across machines",
    ],
  },
];

function TunnelsPage() {
  const qc = useQueryClient();
  const hash = useRouterState({ select: (s) => s.location.hash });
  const { data: tunnels = [], isLoading, isError, error, refetch } = useQuery<Tunnel[]>({
    queryKey: ["tunnels"],
    queryFn: () => apiFetch("/api/tunnels"),
  });

  const [highlight, setHighlight] = useState<string | null>(null);
  useEffect(() => {
    if (!hash || isLoading) return;
    const id = hash.replace(/^#/, "");
    if (id !== "cloudflare" && id !== "tailscale") {
      toast.warning(`Unknown tunnel "${id}" — showing all tunnels.`);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    const el = document.getElementById(id);
    if (!el) {
      toast.warning(`Couldn't locate the ${id} tunnel — showing all tunnels.`);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    el.scrollIntoView({ behavior: "smooth", block: "start" });
    setHighlight(id);
    const timer = setTimeout(() => setHighlight(null), 2200);
    return () => clearTimeout(timer);
  }, [hash, isLoading]);

  const toggle = useMutation({
    mutationFn: ({ type, on }: { type: "cloudflare" | "tailscale"; on: boolean }) =>
      apiFetch(`/api/tunnels/${type}`, { method: on ? "POST" : "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tunnels"] });
      toast.success("Tunnel updated");
    },
    onError: () => toast.error("Couldn't update tunnel"),
  });

  return (
    <div>
      <PageHeader
        title="Tunnels"
        description="Expose the OpenAI-compatible endpoint over a tunnel so devices outside this machine can reach it."
        icon="cloud_sync"
        actions={
          <Button asChild variant="outline">
            <Link to="/endpoint">
              <Icon name="api" size={14} className="mr-1.5" />
              Back to Endpoint
            </Link>
          </Button>
        }
      />

      {isLoading ? (
        <CardsGridSkeleton
          count={2}
          height="h-72"
          className="grid md:grid-cols-2 gap-4"
        />
      ) : isError ? (
        <ErrorState
          title="Couldn’t load tunnels"
          error={error}
          onRetry={() => refetch()}
        />
      ) : (
        <div className="grid md:grid-cols-2 gap-4">
          {META.map((meta) => {
            const t = tunnels.find((x) => x.type === meta.type);
            return (
              <TunnelDetailCard
                key={meta.type}
                meta={meta}
                tunnel={t}
                highlighted={highlight === meta.type}
                onToggle={(on) => toggle.mutate({ type: meta.type, on })}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}

function TunnelDetailCard({
  meta,
  tunnel,
  onToggle,
  highlighted,
}: {
  meta: TunnelMeta;
  tunnel?: Tunnel;
  onToggle: (on: boolean) => void;
  highlighted?: boolean;
}) {
  const enabled = !!tunnel?.is_enabled;
  return (
    <Card
      id={meta.type}
      className={
        "card-elev border-border p-5 flex flex-col gap-4 scroll-mt-20 transition-shadow " +
        (highlighted
          ? "ring-2 ring-brand-600 shadow-elev animate-in fade-in"
          : "")
      }
    >
      <div className="flex items-start gap-3">
        <div
          className={
            "w-11 h-11 rounded-xl bg-surface-2 flex items-center justify-center " + meta.color
          }
        >
          <Icon name={meta.icon} size={24} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <h3 className="font-semibold">{meta.title}</h3>
            <Switch checked={enabled} onCheckedChange={onToggle} />
          </div>
          <p className="text-xs text-text-muted mt-1">{meta.description}</p>
        </div>
      </div>

      <ul className="space-y-1.5">
        {meta.details.map((d) => (
          <li key={d} className="flex items-start gap-2 text-xs text-text-muted">
            <Icon name="check_circle" size={14} className="text-success mt-0.5" />
            <span>{d}</span>
          </li>
        ))}
      </ul>

      <div className="flex items-center justify-between text-xs">
        <StatusBadge variant={enabled ? "success" : "muted"} dot>
          {tunnel?.status ?? "inactive"}
        </StatusBadge>
        <a
          href={meta.docsHref}
          target="_blank"
          rel="noreferrer"
          className="text-brand-600 hover:underline inline-flex items-center gap-1"
        >
          Open docs
          <Icon name="open_in_new" size={12} />
        </a>
      </div>

      {tunnel?.url ? (
        <div className="flex items-center gap-2 bg-surface-2 border border-border rounded-lg px-3 py-2 font-mono text-xs">
          <Input
            readOnly
            value={tunnel.url}
            className="bg-transparent border-0 h-auto p-0 font-mono text-xs focus-visible:ring-0"
          />
          <CopyButton value={tunnel.url} />
        </div>
      ) : (
        <div className="text-xs text-text-muted bg-surface-2 border border-dashed border-border rounded-lg px-3 py-2">
          Toggle on to provision a {meta.type} URL.
        </div>
      )}
    </Card>
  );
}
