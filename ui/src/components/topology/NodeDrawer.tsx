import { Link } from "@tanstack/react-router";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import type {
  ApiKey,
  Combo,
  Connection,
  Provider,
} from "@/lib/mocks/types";
import { ProviderIcon } from "../common/ProviderIcon";
import { StatusBadge } from "../common/StatusBadge";
import { Icon } from "../common/Icon";
import type { TrafficEvent } from "@/lib/mocks/types";
import { formatDistanceToNow } from "date-fns";
import { ListRowsSkeleton } from "../common/Skeletons";
import { DialogQueryState } from "../common/DialogQueryState";
import { Skeleton } from "@/components/ui/skeleton";

export interface SelectedNode {
  kind: "key" | "combo" | "provider";
  id: string;
}

interface Props {
  selected: SelectedNode | null;
  onClose: () => void;
  events: TrafficEvent[];
}

export function NodeDrawer({ selected, onClose, events }: Props) {
  const open = !!selected;

  const providersQ = useQuery({
    queryKey: ["providers"],
    queryFn: () => apiFetch<Provider[]>("/api/providers"),
    enabled: open,
  });
  const connectionsQ = useQuery({
    queryKey: ["connections"],
    queryFn: () => apiFetch<Connection[]>("/api/connections"),
    enabled: open && selected?.kind === "provider",
  });
  const combosQ = useQuery({
    queryKey: ["combos"],
    queryFn: () => apiFetch<Combo[]>("/api/combos"),
    enabled: open,
  });
  const keysQ = useQuery({
    queryKey: ["keys"],
    queryFn: () => apiFetch<ApiKey[]>("/api/keys"),
    enabled: open,
  });

  const providers = providersQ.data ?? [];
  const connections = connectionsQ.data ?? [];
  const combos = combosQ.data ?? [];
  const keys = keysQ.data ?? [];

  const queriesFor = (kind: SelectedNode["kind"]) => {
    if (kind === "provider") return [providersQ, connectionsQ];
    if (kind === "combo") return [combosQ];
    return [keysQ];
  };
  const active = selected ? queriesFor(selected.kind) : [];

  const samples = (() => {
    if (!selected) return [];
    return events
      .filter((e) => {
        if (selected.kind === "provider") return e.provider === selected.id;
        if (selected.kind === "combo") return e.combo_id === selected.id;
        if (selected.kind === "key") return e.api_key_id === selected.id;
        return false;
      })
      .slice(0, 8);
  })();

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="w-[420px] sm:max-w-[420px] overflow-y-auto custom-scrollbar">
        <DialogQueryState
          queries={active}
          errorTitle="Couldn’t load details"
          skeleton={
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <Skeleton className="h-10 w-10 rounded-xl" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-1/2" />
                  <Skeleton className="h-3 w-3/4" />
                </div>
              </div>
              <ListRowsSkeleton rows={4} />
            </div>
          }
        >
          {selected?.kind === "provider" && (
            <ProviderDetail
              provider={providers.find((p) => p.id === selected.id)}
              connections={connections.filter(
                (c) => c.provider === selected.id,
              )}
            />
          )}
          {selected?.kind === "combo" && (
            <ComboDetail combo={combos.find((c) => c.id === selected.id)} />
          )}
          {selected?.kind === "key" && (
            <KeyDetail apiKey={keys.find((k) => k.id === selected.id)} />
          )}
        </DialogQueryState>

        <div className="mt-6">
          <div className="text-xs font-semibold uppercase text-text-muted mb-2">
            Recent Requests
          </div>
          {samples.length === 0 && (
            <div className="text-sm text-text-muted py-6 text-center border border-dashed border-border rounded-lg">
              No samples in current window
            </div>
          )}
          <div className="space-y-1.5">
            {samples.map((s) => (
              <div
                key={s.id}
                className="flex items-center gap-2 p-2 rounded-md bg-surface-2 text-xs"
              >
                <StatusBadge
                  variant={s.status === "success" ? "success" : "danger"}
                  dot
                >
                  {s.status}
                </StatusBadge>
                <div className="flex-1 min-w-0">
                  <div className="font-medium truncate">
                    {s.provider} · {s.model}
                  </div>
                  <div className="text-text-muted">
                    {s.api_key_name} · {formatDistanceToNow(new Date(s.timestamp), { addSuffix: true })}
                  </div>
                </div>
                <div className="text-right tabular-nums">
                  <div>{s.latency_ms}ms</div>
                  <div className="text-text-muted">{s.tokens}t</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

function ProviderDetail({
  provider,
  connections,
}: {
  provider?: Provider;
  connections: Connection[];
}) {
  if (!provider) return null;
  const variant =
    provider.status === "active"
      ? "success"
      : provider.status === "error"
        ? "danger"
        : provider.status === "needs_reauth"
          ? "warning"
          : "muted";
  return (
    <>
      <SheetHeader className="px-0">
        <div className="flex items-center gap-3 mb-1">
          <ProviderIcon provider={provider.id} size={40} />
          <div>
            <SheetTitle>{provider.display_name}</SheetTitle>
            <SheetDescription className="text-xs">
              {provider.description}
            </SheetDescription>
          </div>
        </div>
      </SheetHeader>
      <div className="flex flex-wrap items-center gap-2 mb-4">
        <StatusBadge variant={variant as any} dot>
          {provider.status}
        </StatusBadge>
        {provider.auth_types.map((a) => (
          <StatusBadge key={a} variant="muted">
            {a}
          </StatusBadge>
        ))}
        <Button asChild size="sm" variant="outline" className="ml-auto h-7">
          <Link to="/providers/$id" params={{ id: provider.id }}>
            <Icon name="open_in_new" size={14} className="mr-1" />
            Open page
          </Link>
        </Button>
      </div>
      <div className="text-xs font-semibold uppercase text-text-muted mb-2">
        Connections ({connections.length})
      </div>
      <div className="space-y-1.5">
        {connections.map((c) => (
          <div
            key={c.id}
            className="flex items-center gap-2 p-2 rounded-md border border-border bg-surface text-sm"
          >
            <Icon
              name={c.is_active ? "check_circle" : "cancel"}
              size={16}
              className={c.is_active ? "text-success" : "text-text-muted"}
            />
            <div className="flex-1 min-w-0">
              <div className="font-medium truncate">{c.name}</div>
              <div className="text-xs text-text-muted truncate">
                {c.models.length} model{c.models.length === 1 ? "" : "s"} · {c.auth_type}
              </div>
            </div>
            {c.needs_reauth && (
              <StatusBadge variant="warning" dot>
                reauth
              </StatusBadge>
            )}
          </div>
        ))}
        {connections.length === 0 && (
          <div className="text-sm text-text-muted py-4 text-center border border-dashed border-border rounded-lg">
            No connections
          </div>
        )}
      </div>
    </>
  );
}

function ComboDetail({ combo }: { combo?: Combo }) {
  if (!combo) return null;
  return (
    <>
      <SheetHeader className="px-0">
        <SheetTitle className="flex items-center gap-2">
          <Icon name="layers" className="text-brand-600" />
          {combo.name}
        </SheetTitle>
        <SheetDescription>Strategy: {combo.strategy}</SheetDescription>
      </SheetHeader>
      <div className="mt-4">
        <div className="text-xs font-semibold uppercase text-text-muted mb-2">
          Steps
        </div>
        <div className="space-y-1.5">
          {combo.steps.map((s, i) => (
            <div
              key={i}
              className="flex items-center gap-2 p-2 rounded-md border border-border text-sm"
            >
              <span className="w-5 h-5 rounded-full bg-brand-500/10 text-brand-600 text-xs flex items-center justify-center font-semibold">
                {i + 1}
              </span>
              <ProviderIcon provider={s.provider} size={20} />
              <span className="font-medium">{s.provider}</span>
              <span className="text-text-muted">/ {s.model}</span>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}

function KeyDetail({ apiKey }: { apiKey?: ApiKey }) {
  if (!apiKey) return null;
  return (
    <>
      <SheetHeader className="px-0">
        <SheetTitle className="flex items-center gap-2">
          <Icon name="key" className="text-info" />
          {apiKey.name}
        </SheetTitle>
        <SheetDescription className="font-mono text-xs">
          {apiKey.prefix}…
        </SheetDescription>
      </SheetHeader>
      <div className="grid grid-cols-2 gap-2 mt-4 text-sm">
        <Stat label="RPM" value={apiKey.rpm_limit ?? "—"} />
        <Stat label="TPM" value={apiKey.tpm_limit ?? "—"} />
        <Stat label="Daily cap" value={apiKey.daily_spend_cap ? `$${apiKey.daily_spend_cap}` : "—"} />
        <Stat label="Status" value={apiKey.is_active ? "active" : "inactive"} />
      </div>
      <div className="mt-4">
        <div className="text-xs font-semibold uppercase text-text-muted mb-1">
          Scopes
        </div>
        <div className="flex flex-wrap gap-1">
          {apiKey.scopes.map((s) => (
            <StatusBadge key={s} variant="muted">
              {s}
            </StatusBadge>
          ))}
        </div>
      </div>
    </>
  );
}

function Stat({ label, value }: { label: string; value: any }) {
  return (
    <div className="p-2 rounded-md bg-surface-2">
      <div className="text-[10px] uppercase text-text-muted">{label}</div>
      <div className="font-semibold">{value}</div>
    </div>
  );
}
