import { createFileRoute, Link } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { PageHeader } from "@/components/common/PageHeader";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Icon } from "@/components/common/Icon";
import { StackedListSkeleton, ErrorState } from "@/components/common/Skeletons";
import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { EditConnectionDialog } from "@/components/connections/EditConnectionDialog";
import { toast } from "sonner";
import type { Connection, Provider } from "@/lib/types";
import { formatDistanceToNow } from "date-fns";

export const Route = createFileRoute("/_app/connections")({
  component: ConnectionsPage,
});

type AuthFilter = "all" | "oauth" | "api_key" | "noauth";
type StateFilter = "all" | "active" | "inactive" | "needs_reauth" | "error";

function ConnectionsPage() {
  const qc = useQueryClient();
  const [query, setQuery] = useState("");
  const [auth, setAuth] = useState<AuthFilter>("all");
  const [state, setState] = useState<StateFilter>("all");
  const [toDelete, setToDelete] = useState<Connection | null>(null);
  const [editing, setEditing] = useState<Connection | null>(null);

  const {
    data: conns = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery<Connection[]>({
    queryKey: ["connections"],
    queryFn: () => apiFetch("/api/connections"),
  });
  const providersQ = useQuery<Provider[]>({
    queryKey: ["providers"],
    queryFn: () => apiFetch("/api/providers"),
  });
  const providers = providersQ.data ?? [];

  const providerMap = useMemo(
    () => Object.fromEntries(providers.map((p) => [p.id, p])),
    [providers],
  );

  const filtered = useMemo(() => {
    return conns.filter((c) => {
      if (
        query &&
        !c.name.toLowerCase().includes(query.toLowerCase()) &&
        !c.provider.includes(query.toLowerCase())
      )
        return false;
      if (auth !== "all" && c.auth_type !== auth) return false;
      if (state === "active" && !c.is_active) return false;
      if (state === "inactive" && c.is_active) return false;
      if (state === "needs_reauth" && !c.needs_reauth) return false;
      if (state === "error" && !c.last_error) return false;
      return true;
    });
  }, [conns, query, auth, state]);

  const groups = useMemo(() => {
    const byProv = new Map<string, Connection[]>();
    filtered.forEach((c) => {
      if (!byProv.has(c.provider)) byProv.set(c.provider, []);
      byProv.get(c.provider)!.push(c);
    });
    return Array.from(byProv.entries()).sort(([a], [b]) => a.localeCompare(b));
  }, [filtered]);

  const summary = useMemo(() => {
    return {
      total: conns.length,
      active: conns.filter((c) => c.is_active).length,
      reauth: conns.filter((c) => c.needs_reauth).length,
      error: conns.filter((c) => !!c.last_error).length,
    };
  }, [conns]);

  const toggle = useMutation({
    mutationFn: async ({ id, is_active }: { id: string; is_active: boolean }) =>
      apiFetch(`/api/connections/${id}`, { method: "PUT", body: { is_active } }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["connections"] }),
  });
  const del = useMutation({
    mutationFn: (id: string) => apiFetch(`/api/connections/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["connections"] });
      toast.success("Connection removed");
    },
  });
  const test = useMutation({
    mutationFn: (id: string) => apiFetch(`/api/connections/${id}/test`, { method: "POST" }),
    onSuccess: (r) =>
      toast[r.ok ? "success" : "error"](r.ok ? `OK · ${r.latency_ms}ms` : "Test failed"),
  });
  const bulk = useMutation({
    mutationFn: (op: "enable" | "disable") =>
      apiFetch(`/api/connections/bulk-${op}`, { method: "POST" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["connections"] });
      toast.success("Done");
    },
  });

  return (
    <div>
      <PageHeader
        title="Connections"
        description="Every provider connection across OAuth, API keys and local runtimes."
        icon="link"
        actions={
          <>
            <Button variant="outline" onClick={() => bulk.mutate("disable")}>
              <Icon name="pause" size={16} className="mr-1.5" />
              Pause all
            </Button>
            <Button variant="outline" onClick={() => bulk.mutate("enable")}>
              <Icon name="play_arrow" size={16} className="mr-1.5" />
              Resume all
            </Button>
          </>
        }
      />

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 mb-5">
        <Pill label="Total" value={summary.total} icon="link" />
        <Pill label="Active" value={summary.active} tone="success" icon="check_circle" />
        <Pill label="Needs re-auth" value={summary.reauth} tone="warning" icon="lock_reset" />
        <Pill label="Errors" value={summary.error} tone="danger" icon="error" />
      </div>

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
            placeholder="Search by name or provider…"
            className="pl-9"
          />
        </div>
        <Seg<AuthFilter>
          value={auth}
          onChange={setAuth}
          options={[
            { value: "all", label: "All auth" },
            { value: "oauth", label: "OAuth" },
            { value: "api_key", label: "API key" },
            { value: "noauth", label: "No auth" },
          ]}
        />
        <Seg<StateFilter>
          value={state}
          onChange={setState}
          options={[
            { value: "all", label: "All" },
            { value: "active", label: "Active" },
            { value: "inactive", label: "Inactive" },
            { value: "needs_reauth", label: "Re-auth" },
            { value: "error", label: "Errors" },
          ]}
        />
      </div>

      {isLoading ? (
        <StackedListSkeleton count={4} height="h-32" />
      ) : isError ? (
        <ErrorState
          title="Couldn’t load connections"
          error={error}
          onRetry={() => refetch()}
        />
      ) : groups.length === 0 ? (
        <div className="text-center py-16 text-text-muted text-sm border border-dashed border-border rounded-xl">
          No connections match the current filters.
        </div>
      ) : (
        <div className="space-y-5">
          {groups.map(([pid, list]) => {
            const p = providerMap[pid];
            return (
              <Card key={pid} className="card-elev border-border overflow-hidden">
                <Link
                  to="/providers/$id"
                  params={{ id: pid }}
                  className="flex items-center gap-3 px-4 py-3 bg-surface-2 border-b border-border hover:bg-surface-2/80 transition-colors group"
                >
                  <ProviderIcon provider={pid} size={28} />
                  <div className="min-w-0 flex-1">
                    <div className="font-semibold text-sm truncate flex items-center gap-1.5">
                      {p?.display_name ?? pid}
                      <Icon
                        name="arrow_forward"
                        size={14}
                        className="text-text-muted opacity-0 group-hover:opacity-100 group-hover:translate-x-0.5 transition-all"
                      />
                    </div>
                    <div className="text-[11px] text-text-muted">
                      {list.length} connection{list.length === 1 ? "" : "s"} · Open provider page
                    </div>
                  </div>
                  <StatusBadge
                    variant={
                      p?.status === "active"
                        ? "success"
                        : p?.status === "error"
                          ? "danger"
                          : p?.status === "needs_reauth"
                            ? "warning"
                            : "muted"
                    }
                    dot
                  >
                    {p?.status ?? "unknown"}
                  </StatusBadge>
                </Link>
                <div className="divide-y divide-border">
                  {list.map((c) => (
                    <div
                      key={c.id}
                      className="flex flex-wrap items-center gap-3 px-4 py-3 hover:bg-surface-2/50 transition-colors"
                    >
                      <Switch
                        checked={c.is_active}
                        onCheckedChange={(v) => toggle.mutate({ id: c.id, is_active: v })}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-sm truncate">{c.name}</div>
                        <div className="text-[11px] text-text-muted flex flex-wrap items-center gap-1.5 mt-0.5">
                          <span className="font-mono">{c.auth_type}</span>
                          <span>·</span>
                          <span>
                            {c.models.length} model{c.models.length === 1 ? "" : "s"}
                          </span>
                          {c.expires_at && (
                            <>
                              <span>·</span>
                              <span>
                                renews {formatDistanceToNow(new Date(c.expires_at), { addSuffix: true })}
                              </span>
                            </>
                          )}
                        </div>
                        {c.last_error && (
                          <div className="text-[11px] text-destructive mt-1">
                            <Icon name="error" size={12} className="inline mr-1" />
                            {c.last_error}
                          </div>
                        )}
                      </div>
                      <div className="flex items-center gap-1">
                        {c.needs_reauth && (
                          <StatusBadge variant="warning" dot>
                            Re-auth
                          </StatusBadge>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => test.mutate(c.id)}
                          disabled={test.isPending}
                          title="Test connection"
                        >
                          <Icon name="play_circle" size={16} />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setEditing(c)}
                          title="Edit connection"
                        >
                          <Icon name="edit" size={16} />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setToDelete(c)}
                          title="Remove"
                        >
                          <Icon name="delete" size={16} className="text-destructive" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            );
          })}
        </div>
      )}

      <ConfirmDialog
        open={!!toDelete}
        onOpenChange={(v) => !v && setToDelete(null)}
        title="Remove connection?"
        description={toDelete ? `Disconnect "${toDelete.name}". This cannot be undone.` : ""}
        variant="destructive"
        confirmLabel="Remove"
        onConfirm={() => {
          if (toDelete) del.mutate(toDelete.id);
        }}
      />

      <EditConnectionDialog
        connection={editing}
        provider={editing ? providerMap[editing.provider] ?? null : null}
        open={!!editing}
        onOpenChange={(v) => !v && setEditing(null)}
        onSuccess={() => qc.invalidateQueries({ queryKey: ["connections"] })}
      />
    </div>
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

function Pill({
  label,
  value,
  icon,
  tone,
}: {
  label: string;
  value: number;
  icon: string;
  tone?: "success" | "warning" | "danger";
}) {
  const map: Record<string, string> = {
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
      <div>
        <div className="text-[10px] uppercase tracking-wider text-text-muted">{label}</div>
        <div className="text-lg font-semibold tabular-nums">{value}</div>
      </div>
    </Card>
  );
}
