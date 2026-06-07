import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { QRCodeSVG } from "qrcode.react";
import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { PageHeader } from "@/components/common/PageHeader";
import { CopyButton } from "@/components/common/CopyButton";
import { Icon } from "@/components/common/Icon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { CardsGridSkeleton } from "@/components/common/Skeletons";
import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import type { ApiKey, Tunnel } from "@/lib/mocks/types";
import { useVisibleWindow } from "@/lib/hooks/useVisibleWindow";
import { toast } from "sonner";

export const Route = createFileRoute("/_app/endpoint")({
  component: EndpointPage,
});

type KeyFailure = { action: "copy" | "regenerate" | "revoke" | "enable"; message: string };

function recordAudit(action: string, target: string, details?: string) {
  return apiFetch("/api/audit", { method: "POST", body: { action, target, details } }).catch(
    () => undefined,
  );
}

function EndpointPage() {
  const qc = useQueryClient();
  const [keyToRevoke, setKeyToRevoke] = useState<ApiKey | null>(null);
  const [keyToRegenerate, setKeyToRegenerate] = useState<ApiKey | null>(null);
  const [rowErrors, setRowErrors] = useState<Record<string, KeyFailure | undefined>>({});
  const setRowError = (id: string, f: KeyFailure | undefined) =>
    setRowErrors((prev) => ({ ...prev, [id]: f }));

  const tunnelsQ = useQuery<Tunnel[]>({
    queryKey: ["tunnels"],
    queryFn: () => apiFetch("/api/tunnels"),
  });
  const keysQ = useQuery<ApiKey[]>({
    queryKey: ["keys"],
    queryFn: () => apiFetch("/api/keys"),
  });
  const tunnels = tunnelsQ.data ?? [];
  const tLoading = tunnelsQ.isLoading;
  const keys = keysQ.data ?? [];

  const keysWindow = useVisibleWindow(25, keys.length);
  const visibleKeys = keys.slice(0, keysWindow.visible);

  const [tab, setTab] = useState<"local" | "cloudflare" | "tailscale">("local");
  const localUrl = "http://localhost:8787/v1";

  const cf = tunnels.find((t) => t.type === "cloudflare");
  const ts = tunnels.find((t) => t.type === "tailscale");

  const activeUrl = useMemo(() => {
    if (tab === "cloudflare" && cf?.url) return `${cf.url}/v1`;
    if (tab === "tailscale" && ts?.url) return `${ts.url}/v1`;
    return localUrl;
  }, [tab, cf, ts]);

  const toggleTunnel = useMutation({
    mutationFn: async ({ type, on }: { type: "cloudflare" | "tailscale"; on: boolean }) =>
      apiFetch(`/api/tunnels/${type}`, { method: on ? "POST" : "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tunnels"] });
      toast.success("Tunnel updated");
    },
  });

  const [pendingKeyId, setPendingKeyId] = useState<string | null>(null);
  const toggleKey = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      apiFetch(`/api/keys/${id}`, { method: "PUT", body: { is_active } }),
    onMutate: (vars) => {
      setPendingKeyId(vars.id);
      setRowError(vars.id, undefined);
    },
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ["keys"] });
      toast.success(vars.is_active ? "Key re-enabled" : "Key revoked");
    },
    onError: (e: any, vars) => {
      const msg = e?.message || (vars.is_active ? "Couldn't re-enable key" : "Couldn't revoke key");
      toast.error(msg);
      setRowError(vars.id, { action: vars.is_active ? "enable" : "revoke", message: msg });
    },
    onSettled: () => setPendingKeyId(null),
  });
  const regenerateKey = useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/keys/${id}/regenerate`, { method: "POST" }),
    onMutate: (id) => {
      setPendingKeyId(id);
      setRowError(id, undefined);
    },
    onSuccess: async (k: ApiKey) => {
      qc.invalidateQueries({ queryKey: ["keys"] });
      if (k?.full_key) {
        try {
          await navigator.clipboard.writeText(k.full_key);
          toast.success("Key regenerated — new value copied to clipboard");
        } catch {
          toast.success("Key regenerated", {
            description: "Couldn't auto-copy — copy the new value manually.",
          });
        }
      } else {
        toast.success("Key regenerated");
      }
    },
    onError: (e: any, id) => {
      const msg = e?.message || "Couldn't regenerate key";
      toast.error(msg);
      setRowError(id, { action: "regenerate", message: msg });
    },
    onSettled: () => setPendingKeyId(null),
  });

  const retryFailure = (k: ApiKey) => {
    const f = rowErrors[k.id];
    if (!f) return;
    if (f.action === "regenerate") regenerateKey.mutate(k.id);
    else if (f.action === "revoke") toggleKey.mutate({ id: k.id, is_active: false });
    else if (f.action === "enable") toggleKey.mutate({ id: k.id, is_active: true });
    else if (f.action === "copy") {
      const value = k.full_key ?? k.prefix;
      navigator.clipboard
        .writeText(value)
        .then(() => {
          toast.success(`Copied ${k.name} to clipboard`);
          setRowError(k.id, undefined);
          recordAudit("copy_key", `api_key:${k.name}`);
        })
        .catch(() => toast.error(`Couldn't copy ${k.name}`));
    }
  };

  const exportCsv = () => {
    if (!keys.length) {
      toast.warning("No API keys to export");
      return;
    }
    const escape = (v: unknown) => {
      const s = v == null ? "" : String(v);
      return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
    };
    const generatedAt = new Date().toISOString();
    const headers = [
      "id",
      "name",
      "prefix",
      "status",
      "scopes",
      "rpm_limit",
      "tpm_limit",
      "daily_spend_cap_usd",
      "expires_at",
      "created_at",
      "exported_at",
    ];
    const rows = keys.map((k) =>
      [
        k.id,
        k.name,
        k.prefix,
        k.is_active ? "active" : "revoked",
        (k.scopes ?? []).join("|"),
        k.rpm_limit ?? "",
        k.tpm_limit ?? "",
        k.daily_spend_cap ?? "",
        k.expires_at ?? "",
        k.created_at,
        generatedAt,
      ]
        .map(escape)
        .join(","),
    );
    const csv = [headers.join(","), ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `api-keys-${generatedAt.replace(/[:.]/g, "-")}.csv`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    toast.success(`Exported ${keys.length} API key${keys.length === 1 ? "" : "s"}`);
    recordAudit("export_keys", "api_keys", `count=${keys.length}`);
  };

  const sampleCurl = `curl ${activeUrl}/chat/completions \\
  -H "Authorization: Bearer ${keys[0]?.prefix ?? "sk-…"}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hi"}]}'`;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Endpoint"
        description="One OpenAI-compatible URL for every connected model. Share it locally, over the internet, or on your tailnet."
        icon="api"
      />

      {/* Hero URL block */}
      <Card className="card-elev border-border overflow-hidden">
        <div className="grid lg:grid-cols-[1fr_auto] gap-6 p-6">
          <div className="min-w-0 space-y-4">
            <div className="flex items-center gap-1 bg-surface-2 rounded-lg p-1 w-fit">
              <SegBtn active={tab === "local"} onClick={() => setTab("local")} icon="computer">
                Local
              </SegBtn>
              <SegBtn
                active={tab === "cloudflare"}
                onClick={() => setTab("cloudflare")}
                icon="cloud"
                disabled={!cf?.is_enabled}
              >
                Cloudflare
              </SegBtn>
              <SegBtn
                active={tab === "tailscale"}
                onClick={() => setTab("tailscale")}
                icon="hub"
                disabled={!ts?.is_enabled}
              >
                Tailscale
              </SegBtn>
            </div>

            <div>
              <div className="text-xs uppercase tracking-wider text-text-muted mb-1.5 flex items-center gap-2">
                <Icon name="link" size={14} />
                Public endpoint
              </div>
              <div className="flex items-center gap-2 bg-surface-2 border border-border rounded-xl px-4 py-3 font-mono text-base">
                <span className="flex-1 truncate select-all">{activeUrl}</span>
                <CopyButton value={activeUrl} variant="outline" />
              </div>
              <p className="text-xs text-text-muted mt-2">
                {tab === "local" && "Reachable from this machine and devices on your LAN."}
                {tab === "cloudflare" && (cf?.is_enabled ? "Reachable from anywhere on the internet." : "Enable the Cloudflare tunnel below to expose this URL publicly.")}
                {tab === "tailscale" && (ts?.is_enabled ? "Reachable from your tailnet only." : "Enable Tailscale below to expose this URL on your tailnet.")}
              </p>
            </div>

            <div>
              <div className="text-xs uppercase tracking-wider text-text-muted mb-1.5 flex items-center gap-2">
                <Icon name="terminal" size={14} />
                Quick start
              </div>
              <pre className="bg-surface-2 border border-border rounded-xl p-4 text-xs font-mono overflow-x-auto leading-relaxed relative">
                <CopyButton
                  value={sampleCurl}
                  className="absolute top-2 right-2"
                />
                <code>{sampleCurl}</code>
              </pre>
            </div>
          </div>

          <div className="flex flex-col items-center justify-center gap-3 lg:border-l lg:border-border lg:pl-6">
            <div className="bg-white p-3 rounded-xl shadow-elev">
              <QRCodeSVG value={activeUrl} size={144} level="M" />
            </div>
            <div className="text-xs text-text-muted text-center max-w-[160px]">
              Scan to copy the endpoint to a phone or another device.
            </div>
          </div>
        </div>
      </Card>

      {/* Tunnels grid */}
      <div>
        <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wider mb-3">
          Tunnels
        </h2>
        <div className="grid md:grid-cols-2 gap-4">
          {tLoading ? (
            <CardsGridSkeleton
              count={2}
              height="h-44"
              className="contents"
            />
          ) : (
            <>
              <TunnelCard
                title="Cloudflare Tunnel"
                description="Expose the endpoint over a trycloudflare.com URL. Zero config, no port forwarding."
                icon="cloud"
                color="text-info"
                tunnel={cf}
                onToggle={(on) => toggleTunnel.mutate({ type: "cloudflare", on })}
                docsHref="https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/"
              />
              <TunnelCard
                title="Tailscale"
                description="Reachable only from devices on your tailnet. Best for teams that want a private endpoint."
                icon="hub"
                color="text-brand-600"
                tunnel={ts}
                onToggle={(on) => toggleTunnel.mutate({ type: "tailscale", on })}
                docsHref="https://tailscale.com/kb/1080/cli"
              />
            </>
          )}
        </div>
      </div>

      {/* API keys */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wider">
            API keys
          </h2>
          <Button
            variant="outline"
            size="sm"
            onClick={exportCsv}
            disabled={!keys.length}
            title="Download API keys as CSV"
          >
            <Icon name="download" size={14} className="mr-1.5" />
            Export keys
          </Button>
        </div>
        <Card className="border-border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-surface-2 text-[11px] uppercase tracking-wider text-text-muted text-left">
              <tr>
                <th className="px-4 py-2">Name</th>
                <th className="px-4 py-2">Prefix</th>
                <th className="px-4 py-2">RPM</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2" />
              </tr>
            </thead>
            <tbody>
              {visibleKeys.map((k) => {
                const isPending = pendingKeyId === k.id;
                const failure = rowErrors[k.id];
                return (
                <tr key={k.id} className="border-t border-border align-top">
                  <td className="px-4 py-2 font-medium">{k.name}</td>
                  <td className="px-4 py-2 font-mono text-xs">{k.prefix}…</td>
                  <td className="px-4 py-2 text-xs text-text-muted">{k.rpm_limit ?? "—"}</td>
                  <td className="px-4 py-2">
                    <StatusBadge variant={k.is_active ? "success" : "muted"} dot>
                      {k.is_active ? "active" : "revoked"}
                    </StatusBadge>
                  </td>
                  <td className="px-4 py-2 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <CopyButton
                        value={k.full_key ?? k.prefix}
                        label="Copy"
                        disabled={isPending}
                        successMessage={`Copied ${k.name} to clipboard`}
                        errorMessage={`Couldn't copy ${k.name}`}
                        onSuccess={() => {
                          setRowError(k.id, undefined);
                          recordAudit("copy_key", `api_key:${k.name}`);
                        }}
                        onError={(err: any) =>
                          setRowError(k.id, {
                            action: "copy",
                            message: err?.message || `Couldn't copy ${k.name}`,
                          })
                        }
                      />

                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setKeyToRegenerate(k)}
                        disabled={isPending}
                        title="Regenerate key — invalidates the old value"
                      >
                        <Icon
                          name={isPending && regenerateKey.isPending ? "hourglass_empty" : "autorenew"}
                          size={14}
                          className={
                            "mr-1 " +
                            (isPending && regenerateKey.isPending ? "animate-spin" : "")
                          }
                        />
                        {isPending && regenerateKey.isPending ? "Regenerating…" : "Regenerate"}
                      </Button>
                      {k.is_active ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setKeyToRevoke(k)}
                          disabled={isPending}
                          title="Revoke key"
                        >
                          <Icon
                            name={isPending && toggleKey.isPending ? "hourglass_empty" : "block"}
                            size={14}
                            className={
                              "mr-1 " +
                              (isPending && toggleKey.isPending
                                ? "animate-spin"
                                : "text-destructive")
                            }
                          />
                          {isPending && toggleKey.isPending ? "Revoking…" : "Revoke"}
                        </Button>
                      ) : (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => toggleKey.mutate({ id: k.id, is_active: true })}
                          disabled={isPending}
                          title="Re-enable key"
                        >
                          <Icon
                            name={isPending && toggleKey.isPending ? "hourglass_empty" : "restart_alt"}
                            size={14}
                            className={
                              "mr-1 " +
                              (isPending && toggleKey.isPending ? "animate-spin" : "")
                            }
                          />
                          {isPending && toggleKey.isPending ? "Enabling…" : "Enable"}
                        </Button>
                      )}
                    </div>
                    {failure && !isPending && (
                      <div className="mt-2 flex items-center justify-end gap-2 text-[11px] text-destructive">
                        <Icon name="error" size={12} />
                        <span className="max-w-[260px] truncate" title={failure.message}>
                          {failure.action === "copy"
                            ? "Copy failed"
                            : failure.action === "regenerate"
                              ? "Regenerate failed"
                              : failure.action === "revoke"
                                ? "Revoke failed"
                                : "Enable failed"}
                          : {failure.message}
                        </span>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 px-2 text-destructive hover:text-destructive"
                          onClick={() => retryFailure(k)}
                        >
                          <Icon name="refresh" size={12} className="mr-1" />
                          Retry
                        </Button>
                      </div>
                    )}
                  </td>
                </tr>
                );
              })}
              {!keys.length && (
                <tr>
                  <td colSpan={5} className="py-6 text-center text-text-muted text-sm">
                    No API keys yet
                  </td>
                </tr>
              )}
            </tbody>
          </table>
          {keysWindow.hasMore && (
            <div
              ref={keysWindow.sentinelRef}
              className="flex items-center justify-between border-t border-border bg-surface-2/30 px-4 py-2 text-xs text-text-muted"
            >
              <span>
                Showing {visibleKeys.length} of {keys.length}
              </span>
              <Button variant="outline" size="sm" onClick={keysWindow.loadMore}>
                Load more
              </Button>
            </div>
          )}
        </Card>
      </div>

      <ConfirmDialog
        open={!!keyToRevoke}
        onOpenChange={(v) => !v && setKeyToRevoke(null)}
        title="Revoke API key?"
        description={
          keyToRevoke
            ? `"${keyToRevoke.name}" will stop working immediately. You can re-enable it later.`
            : ""
        }
        variant="destructive"
        confirmLabel="Revoke"
        onConfirm={() => {
          if (keyToRevoke) toggleKey.mutate({ id: keyToRevoke.id, is_active: false });
          setKeyToRevoke(null);
        }}
      />
      <ConfirmDialog
        open={!!keyToRegenerate}
        onOpenChange={(v) => !v && setKeyToRegenerate(null)}
        title="Regenerate API key?"
        description={
          keyToRegenerate
            ? `The previous value of "${keyToRegenerate.name}" stops working immediately. The new value is copied to your clipboard.`
            : ""
        }
        variant="destructive"
        confirmLabel="Regenerate"
        onConfirm={() => {
          if (keyToRegenerate) regenerateKey.mutate(keyToRegenerate.id);
          setKeyToRegenerate(null);
        }}
      />
    </div>
  );
}

function SegBtn({
  active,
  onClick,
  icon,
  disabled,
  children,
}: {
  active: boolean;
  onClick: () => void;
  icon: string;
  disabled?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={
        "px-3 py-1.5 text-xs rounded-md font-medium inline-flex items-center gap-1.5 transition-colors " +
        (active
          ? "bg-surface text-foreground shadow-soft"
          : "text-text-muted hover:text-foreground disabled:opacity-40 disabled:cursor-not-allowed")
      }
    >
      <Icon name={icon} size={14} />
      {children}
    </button>
  );
}

function TunnelCard({
  title,
  description,
  icon,
  color,
  tunnel,
  onToggle,
  docsHref,
}: {
  title: string;
  description: string;
  icon: string;
  color: string;
  tunnel?: Tunnel;
  onToggle: (on: boolean) => void;
  docsHref: string;
}) {
  const enabled = !!tunnel?.is_enabled;
  return (
    <Card className="card-elev border-border p-5 flex flex-col gap-4">
      <div className="flex items-start gap-3">
        <div className={"w-10 h-10 rounded-lg bg-surface-2 flex items-center justify-center " + color}>
          <Icon name={icon} size={22} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <h3 className="font-semibold">{title}</h3>
            <Switch checked={enabled} onCheckedChange={onToggle} />
          </div>
          <p className="text-xs text-text-muted mt-1">{description}</p>
        </div>
      </div>
      <div className="flex items-center justify-between text-xs">
        <StatusBadge variant={enabled ? "success" : "muted"} dot>
          {tunnel?.status ?? "inactive"}
        </StatusBadge>
        <div className="flex items-center gap-3">
          <a
            href={docsHref}
            target="_blank"
            rel="noreferrer"
            className="text-brand-600 hover:underline inline-flex items-center gap-1"
          >
            Docs
            <Icon name="open_in_new" size={12} />
          </a>
          <Link
            to="/tunnels"
            hash={tunnel?.type}
            className="text-brand-600 hover:underline inline-flex items-center gap-1"
          >
            Manage
            <Icon name="arrow_forward" size={12} />
          </Link>
        </div>
      </div>
      {tunnel?.url && (
        <div className="flex items-center gap-2 bg-surface-2 border border-border rounded-lg px-3 py-2 font-mono text-xs">
          <Input
            readOnly
            value={tunnel.url}
            className="bg-transparent border-0 h-auto p-0 font-mono text-xs focus-visible:ring-0"
          />
          <CopyButton value={tunnel.url} />
        </div>
      )}
    </Card>
  );
}
