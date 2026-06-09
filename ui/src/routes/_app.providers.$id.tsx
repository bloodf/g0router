import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PageHeader } from "@/components/common/PageHeader";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Icon } from "@/components/common/Icon";
import { ProviderTopology } from "@/components/topology/ProviderTopology";
import { DetailHeaderSkeleton, TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { EditConnectionDialog } from "@/components/connections/EditConnectionDialog";
import type { Connection, Model, Provider } from "@/lib/types";
import { toast } from "sonner";

export const Route = createFileRoute("/_app/providers/$id")({
  component: ProviderDetail,
});

function ProviderDetail() {
  const { id } = Route.useParams();
  const qc = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Connection | null>(null);
  const [toDelete, setToDelete] = useState<Connection | null>(null);
  const [suggestedOpen, setSuggestedOpen] = useState(false);

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
  const { data: models = [] } = useQuery<Model[]>({
    queryKey: ["provider", id, "models"],
    queryFn: () => apiFetch(`/api/providers/${id}/models`),
  });
  const {
    data: suggested = [],
    isFetching: suggestedFetching,
    isError: suggestedError,
    error: suggestedErr,
  } = useQuery<{ id: string; name: string }[]>({
    queryKey: ["provider", id, "suggested-models"],
    queryFn: () => apiFetch(`/api/providers/${id}/suggested-models`),
    enabled: suggestedOpen,
  });

  const del = useMutation({
    mutationFn: (cid: string) => apiFetch(`/api/connections/${cid}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["provider", id, "connections"] });
      qc.invalidateQueries({ queryKey: ["connections"] });
      refetchProvider();
      toast.success("Connection removed");
    },
  });

  const testConn = async (cid: string) => {
    const r = await apiFetch(`/api/connections/${cid}/test`, { method: "POST" });
    toast[r.ok ? "success" : "error"](
      r.ok ? `OK (${r.latency_ms}ms)` : "Failed",
    );
  };

  if (providerError)
    return (
      <ErrorState
        title="Couldn't load provider"
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
          <ProviderIcon provider={provider.id} iconUrl={provider.icon_url} size={56} />
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
        <Button className="btn-cta" onClick={() => setCreateOpen(true)}>
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

        <TabsContent value="overview" className="mt-4 space-y-4">
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

          <Card className="p-6 card-elev border-border">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-semibold">Suggested models</h3>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSuggestedOpen(true)}
                disabled={suggestedFetching}
              >
                <Icon name="auto_awesome" size={16} className="mr-1.5" />
                {suggestedFetching ? "Loading..." : "Load suggestions"}
              </Button>
            </div>
            {suggestedError ? (
              <div className="text-sm text-destructive">
                Failed to load suggestions: {String((suggestedErr as any)?.message || suggestedErr)}
              </div>
            ) : suggestedOpen && !suggestedFetching ? (
              <div className="flex flex-wrap gap-2">
                {suggested.length > 0 ? (
                  suggested.map((m) => (
                    <StatusBadge key={m.id} variant="muted">
                      {m.name || m.id}
                    </StatusBadge>
                  ))
                ) : (
                  <span className="text-sm text-text-muted">No suggestions available.</span>
                )}
              </div>
            ) : (
              <p className="text-sm text-text-muted">
                Fetch the latest model list from the provider using an active connection.
              </p>
            )}
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
                      <Button variant="ghost" size="sm" onClick={() => testConn(c.id)} title="Test connection">
                        <Icon name="play_circle" size={14} />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => setEditing(c)} title="Edit connection">
                        <Icon name="edit" size={14} />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => setToDelete(c)} title="Remove connection">
                        <Icon name="delete" size={14} className="text-destructive" />
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

      <CreateConnectionDialog
        provider={provider}
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={() => {
          refetchConn();
          refetchProvider();
          qc.invalidateQueries({ queryKey: ["connections"] });
        }}
      />

      <EditConnectionDialog
        key={editing?.id ?? "closed"}
        connection={editing}
        provider={provider}
        open={!!editing}
        onOpenChange={(v) => !v && setEditing(null)}
        onSuccess={() => {
          refetchConn();
          refetchProvider();
          qc.invalidateQueries({ queryKey: ["connections"] });
        }}
      />

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
    </div>
  );
}

function CreateConnectionDialog({
  provider,
  open,
  onOpenChange,
  onSuccess,
}: {
  provider: Provider;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSuccess: () => void;
}) {
  const [name, setName] = useState("");
  const [authType, setAuthType] = useState(provider.auth_types[0] ?? "api_key");
  const [credential, setCredential] = useState("");
  const [isActive, setIsActive] = useState(true);
  const [busy, setBusy] = useState(false);

  const credentialLabel = useMemo(() => {
    if (authType === "api_key") return "API key";
    if (authType === "oauth") return "Access token";
    return undefined;
  }, [authType]);

  const needsCredential = authType === "api_key" || authType === "oauth";

  const reset = () => {
    setName("");
    setAuthType(provider.auth_types[0] ?? "api_key");
    setCredential("");
    setIsActive(true);
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Connection name is required");
      return;
    }
    if (needsCredential && !credential.trim()) {
      toast.error(`${credentialLabel} is required`);
      return;
    }

    const body: Record<string, unknown> = {
      provider: provider.id,
      name: name.trim(),
      auth_type: authType,
      is_active: isActive,
    };
    if (authType === "api_key" && credential.trim()) {
      body.api_key = credential.trim();
    }
    if (authType === "oauth" && credential.trim()) {
      body.access_token = credential.trim();
    }

    setBusy(true);
    try {
      await apiFetch("/api/connections", {
        method: "POST",
        body,
      });
      toast.success("Connection created");
      reset();
      onOpenChange(false);
      onSuccess();
    } catch (err: any) {
      toast.error(err?.message || "Failed to create connection");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ProviderIcon provider={provider.id} iconUrl={provider.icon_url} size={24} />
            New connection — {provider.display_name}
          </DialogTitle>
          <DialogDescription>
            Add credentials to connect {provider.display_name} to g0router.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="conn-name">Name</Label>
            <Input
              id="conn-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Primary"
              autoFocus
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="conn-auth">Auth type</Label>
            <Select value={authType} onValueChange={(v) => setAuthType(v as any)}>
              <SelectTrigger id="conn-auth" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {provider.auth_types.map((a) => (
                  <SelectItem key={a} value={a}>
                    {a.replace("_", " ")}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {needsCredential && (
            <div className="space-y-1.5">
              <Label htmlFor="conn-credential">{credentialLabel}</Label>
              <Input
                id="conn-credential"
                type="password"
                value={credential}
                onChange={(e) => setCredential(e.target.value)}
                placeholder={credentialLabel}
              />
            </div>
          )}

          <div className="flex items-center justify-between">
            <Label htmlFor="conn-active" className="cursor-pointer">Active</Label>
            <Switch
              id="conn-active"
              checked={isActive}
              onCheckedChange={setIsActive}
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
              Cancel
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? "Creating..." : "Create connection"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
