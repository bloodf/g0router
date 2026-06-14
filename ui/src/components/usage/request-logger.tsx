import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import { Badge } from "@/components/ui/badge";
import { ProviderIcon } from "@/components/ui/provider-icon";
import { CardSkeleton } from "@/components/ui/skeleton";
import type { UsageLog } from "@/lib/types";

// A normalized display row, derived from either the real Go pipe-delimited
// string (internal/usage/logs.go:41) or the structured UsageLog object.
export interface LogRow {
  timestamp: string;
  provider: string;
  model: string;
  status: string;
  status_code?: number;
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
  cost_usd?: number;
  latency_ms?: number;
}

function isUsageLog(v: unknown): v is UsageLog {
  return typeof v === "object" && v !== null && "model" in v && "provider" in v;
}

// normalizeLogRow accepts both shapes (plan §1.4 real-vs-mock tolerance, same
// client-side normalization precedent as connections.tsx). Real Go format:
//   "ts | model | PROVIDER | account | sent | received | status"
export function normalizeLogRow(raw: UsageLog | string): LogRow {
  if (isUsageLog(raw)) {
    return {
      timestamp: raw.timestamp,
      provider: raw.provider,
      model: raw.model,
      status: raw.status,
      status_code: raw.status_code,
      prompt_tokens: raw.prompt_tokens,
      completion_tokens: raw.completion_tokens,
      total_tokens: raw.total_tokens,
      cost_usd: raw.cost_usd,
      latency_ms: raw.latency_ms,
    };
  }
  const parts = String(raw).split("|").map((s) => s.trim());
  const [ts = "-", model = "-", provider = "-", , sent = "-", received = "-", status = "-"] = parts;
  const toNum = (s: string) => {
    const n = Number(s);
    return Number.isFinite(n) ? n : undefined;
  };
  return {
    timestamp: ts,
    provider,
    model,
    status,
    prompt_tokens: toNum(sent),
    completion_tokens: toNum(received),
  };
}

async function fetchLogs(): Promise<LogRow[]> {
  const data = await apiFetch<(UsageLog | string)[]>("/api/usage/request-logs");
  return (data ?? []).map(normalizeLogRow);
}

// startLogPolling drives the 3s auto-refresh (PAR-UI-048 / PAR-USAGE-037). Pure
// (no React) so the interval contract is unit-provable with fake timers.
export function startLogPolling(fetchFn: () => unknown, intervalMs = 3000): () => void {
  const id = setInterval(() => {
    fetchFn();
  }, intervalMs);
  return () => clearInterval(id);
}

function statusVariant(row: LogRow): "success" | "error" | "neutral" {
  if (row.status === "success" || (row.status_code && row.status_code < 400)) return "success";
  if (row.status === "error" || (row.status_code && row.status_code >= 400)) return "error";
  return "neutral";
}

export interface RequestLoggerProps {
  initialLogs?: (UsageLog | string)[];
  compact?: boolean;
}

export function RequestLogger({ initialLogs, compact }: RequestLoggerProps) {
  const [rows, setRows] = React.useState<LogRow[]>(
    initialLogs ? initialLogs.map(normalizeLogRow) : [],
  );
  const [loading, setLoading] = React.useState(!initialLogs);
  const [autoRefresh, setAutoRefresh] = React.useState(true);

  const load = React.useCallback(() => {
    fetchLogs()
      .then((logs) => {
        setRows(logs);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  React.useEffect(() => {
    load();
  }, [load]);

  React.useEffect(() => {
    if (!autoRefresh) return;
    return startLogPolling(load, 3000);
  }, [autoRefresh, load]);

  if (loading && rows.length === 0) {
    return <CardSkeleton />;
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Toggle checked={autoRefresh} onCheckedChange={setAutoRefresh} aria-label="Auto refresh" />
          <span>Auto-refresh (3s)</span>
        </div>
        <Button variant="outline" size="sm" data-testid="logs-refresh" onClick={load}>
          Refresh
        </Button>
      </div>
      <div className="overflow-x-auto rounded-xl border border-border bg-card">
        <table className="w-full text-sm" data-testid="request-log-table">
          <thead>
            <tr className="border-b border-border text-left text-muted-foreground">
              <th className="px-4 py-2 font-medium">Time</th>
              <th className="px-4 py-2 font-medium">Provider</th>
              <th className="px-4 py-2 font-medium">Model</th>
              <th className="px-4 py-2 font-medium">Status</th>
              {!compact ? <th className="px-4 py-2 font-medium">Tokens</th> : null}
              {!compact ? <th className="px-4 py-2 font-medium">Cost</th> : null}
              {!compact ? <th className="px-4 py-2 font-medium">Latency</th> : null}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => (
              <tr key={`${row.timestamp}-${i}`} className="border-b border-border/50">
                <td className="whitespace-nowrap px-4 py-2 text-muted-foreground">{row.timestamp}</td>
                <td className="px-4 py-2">
                  <span className="inline-flex items-center gap-2">
                    <ProviderIcon slug={row.provider.toLowerCase()} name={row.provider} size="sm" />
                    {row.provider}
                  </span>
                </td>
                <td className="px-4 py-2 text-foreground">{row.model}</td>
                <td className="px-4 py-2">
                  <Badge variant={statusVariant(row)}>{row.status}</Badge>
                </td>
                {!compact ? (
                  <td className="px-4 py-2">
                    {row.total_tokens ?? (row.prompt_tokens ?? 0) + (row.completion_tokens ?? 0)}
                  </td>
                ) : null}
                {!compact ? <td className="px-4 py-2">{row.cost_usd !== undefined ? `$${row.cost_usd.toFixed(4)}` : "-"}</td> : null}
                {!compact ? <td className="px-4 py-2">{row.latency_ms !== undefined ? `${row.latency_ms}ms` : "-"}</td> : null}
              </tr>
            ))}
            {rows.length === 0 ? (
              <tr>
                <td colSpan={compact ? 4 : 7} className="px-4 py-6 text-center text-muted-foreground">
                  No requests yet.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
