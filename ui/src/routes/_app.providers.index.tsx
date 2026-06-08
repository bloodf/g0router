import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/PageHeader";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Icon } from "@/components/common/Icon";
import { CardsGridSkeleton, ErrorState } from "@/components/common/Skeletons";
import { toast } from "sonner";
import type { Provider } from "@/lib/types";

export const Route = createFileRoute("/_app/providers/")({
  component: ProvidersPage,
});

type StatusFilter = "all" | "active" | "needs_reauth" | "error" | "inactive";
type AuthFilter = "all" | "oauth" | "api_key" | "noauth" | "custom";

function ProvidersPage() {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [auth, setAuth] = useState<AuthFilter>("all");
  const [onlyConnected, setOnlyConnected] = useState(false);

  const { data: providers = [], isLoading, isError, error, refetch } = useQuery<Provider[]>({
    queryKey: ["providers"],
    queryFn: () => apiFetch("/api/providers"),
  });

  const summary = useMemo(() => {
    return {
      total: providers.length,
      active: providers.filter((p) => p.status === "active").length,
      reauth: providers.filter((p) => p.status === "needs_reauth").length,
      error: providers.filter((p) => p.status === "error").length,
      connected: providers.filter((p) => p.connection_count > 0).length,
    };
  }, [providers]);

  const filtered = useMemo(() => {
    return providers.filter((p) => {
      if (
        query &&
        !p.display_name.toLowerCase().includes(query.toLowerCase()) &&
        !p.id.includes(query.toLowerCase())
      )
        return false;
      if (onlyConnected && p.connection_count === 0) return false;
      if (status !== "all" && p.status !== status) return false;
      if (auth !== "all" && !p.auth_types.includes(auth)) return false;
      return true;
    });
  }, [providers, query, status, auth, onlyConnected]);

  const grouped = useMemo(() => {
    const connected = filtered.filter((p) => p.connection_count > 0);
    const available = filtered.filter((p) => p.connection_count === 0);
    return { connected, available };
  }, [filtered]);

  const batchTest = async () => {
    const r = await apiFetch("/api/providers/test-batch", { method: "POST" });
    const ok = r.results.filter((x: any) => x.ok).length;
    toast.success(`Tested ${r.results.length} — ${ok} passed`);
  };

  return (
    <div>
      <PageHeader
        title="Providers"
        description="Connect to 40+ LLM providers via OAuth, API key, or local runtime."
        icon="dns"
        actions={
          <>
            <Button variant="outline" onClick={batchTest}>
              <Icon name="check_circle" size={16} className="mr-1.5" />
              Test all
            </Button>
            <Button onClick={() => refetch()}>
              <Icon name="refresh" size={16} className="mr-1.5" />
              Refresh
            </Button>
          </>
        }
      />

      {/* Summary strip */}
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-2 mb-5">
        <Stat label="Total" value={summary.total} icon="dns" />
        <Stat label="Connected" value={summary.connected} icon="link" tone="info" />
        <Stat label="Active" value={summary.active} icon="check_circle" tone="success" />
        <Stat label="Needs re-auth" value={summary.reauth} icon="lock_reset" tone="warning" />
        <Stat label="Errors" value={summary.error} icon="error" tone="danger" />
      </div>

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2 mb-5">
        <div className="relative flex-1 min-w-[220px] max-w-md">
          <Icon
            name="search"
            size={16}
            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted"
          />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search providers…"
            className="pl-9"
          />
        </div>
        <Seg<StatusFilter>
          value={status}
          onChange={setStatus}
          options={[
            { value: "all", label: "All status" },
            { value: "active", label: "Active" },
            { value: "needs_reauth", label: "Re-auth" },
            { value: "error", label: "Error" },
            { value: "inactive", label: "Inactive" },
          ]}
        />
        <Seg<AuthFilter>
          value={auth}
          onChange={setAuth}
          options={[
            { value: "all", label: "All auth" },
            { value: "oauth", label: "OAuth" },
            { value: "api_key", label: "API key" },
            { value: "noauth", label: "No auth" },
            { value: "custom", label: "Custom" },
          ]}
        />
        <label className="inline-flex items-center gap-2 text-xs text-text-muted cursor-pointer ml-1">
          <input
            type="checkbox"
            className="accent-brand-500 w-3.5 h-3.5"
            checked={onlyConnected}
            onChange={(e) => setOnlyConnected(e.target.checked)}
          />
          Connected only
        </label>
      </div>

      {isLoading ? (
        <CardsGridSkeleton count={8} height="h-36" />
      ) : isError ? (
        <ErrorState
          title="Couldn\u2019t load providers"
          error={error}
          onRetry={() => refetch()}
        />
      ) : (
        <div className="space-y-8">
          <Section
            title="Connected"
            count={grouped.connected.length}
            providers={grouped.connected}
          />
          <Section
            title="Available"
            count={grouped.available.length}
            providers={grouped.available}
            muted
          />
          {!filtered.length && (
            <div className="text-center py-16 text-text-muted text-sm border border-dashed border-border rounded-xl">
              No providers match the current filters.
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function Section({
  title,
  count,
  providers,
  muted,
}: {
  title: string;
  count: number;
  providers: Provider[];
  muted?: boolean;
}) {
  if (!providers.length) return null;
  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-text-muted">
          {title}
          <span className="ml-2 text-text-muted/70">({count})</span>
        </h2>
      </div>
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
        {providers.map((p) => (
          <ProviderCard key={p.id} provider={p} muted={muted} />
        ))}
      </div>
    </div>
  );
}

function ProviderCard({ provider: p, muted }: { provider: Provider; muted?: boolean }) {
  return (
    <Link to="/providers/$id" params={{ id: p.id }} className="block group">
      <Card
        className={
          "p-4 card-elev border-border h-full transition-all group-hover:shadow-warm group-hover:-translate-y-0.5 " +
          (muted ? "opacity-80 hover:opacity-100" : "")
        }
      >
        <div className="flex items-start gap-3">
          <ProviderIcon provider={p.id} size={40} />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <div className="font-semibold truncate">{p.display_name}</div>
              <StatusBadge
                variant={
                  p.status === "active"
                    ? "success"
                    : p.status === "needs_reauth"
                      ? "warning"
                      : p.status === "error"
                        ? "danger"
                        : "muted"
                }
                dot
                className="ml-auto"
              >
                {p.connection_count}
              </StatusBadge>
            </div>
            <div className="flex flex-wrap items-center gap-1 mt-1.5">
              {p.auth_types.map((a) => (
                <StatusBadge key={a} variant="primary">
                  {a.replace("_", " ")}
                </StatusBadge>
              ))}
            </div>
          </div>
        </div>
        <p className="text-xs text-text-muted mt-3 line-clamp-2">{p.description}</p>
        <div className="flex items-center justify-between mt-3 pt-3 border-t border-border text-[11px] text-text-muted">
          <div className="flex flex-wrap gap-1">
            {p.capabilities.slice(0, 3).map((c) => (
              <span key={c} className="px-1.5 py-0.5 rounded bg-surface-2">
                {c}
              </span>
            ))}
            {p.capabilities.length > 3 && (
              <span className="px-1.5 py-0.5 rounded bg-surface-2">
                +{p.capabilities.length - 3}
              </span>
            )}
          </div>
          <Icon
            name="arrow_forward"
            size={14}
            className="text-text-muted group-hover:text-brand-600 group-hover:translate-x-0.5 transition-transform"
          />
        </div>
      </Card>
    </Link>
  );
}

function Seg<T extends string>({
  value,
  onChange,
  options,
}: {
  value: T;
  onChange: (v: T) => void;
  options: { value: T; label: string }[];
}) {
  return (
    <div className="flex items-center gap-1 bg-surface-2 rounded-lg p-1">
      {options.map((o) => (
        <button
          key={o.value}
          onClick={() => onChange(o.value)}
          className={
            "px-2.5 py-1 text-xs rounded-md transition-colors " +
            (value === o.value
              ? "bg-surface text-foreground shadow-soft font-medium"
              : "text-text-muted hover:text-foreground")
          }
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

function Stat({
  label,
  value,
  icon,
  tone,
}: {
  label: string;
  value: number;
  icon: string;
  tone?: "info" | "success" | "warning" | "danger";
}) {
  const map: Record<string, string> = {
    info: "text-info bg-info/10",
    success: "text-success bg-success/10",
    warning: "text-warning bg-warning/10",
    danger: "text-destructive bg-destructive/10",
  };
  return (
    <Card className="card-elev border-border p-3 flex items-center gap-3">
      <div
        className={
          "w-9 h-9 rounded-lg flex items-center justify-center " +
          (tone ? map[tone] : "bg-surface-2 text-text-muted")
        }
      >
        <Icon name={icon} size={18} />
      </div>
      <div className="min-w-0">
        <div className="text-[10px] uppercase tracking-wider text-text-muted">{label}</div>
        <div className="text-lg font-semibold tabular-nums">{value}</div>
      </div>
    </Card>
  );
}
