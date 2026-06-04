import { useEffect, useState } from "react";
import { apiFetch, isAuthExpiredError, type UsageListResponse, type UsageLogRecord } from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

type LogsState =
  | { status: "loading" }
  | { status: "success"; data: UsageListResponse }
  | { status: "empty"; data: UsageListResponse }
  | { status: "error"; message: string }
  | { status: "auth-expired"; message: string };

export function LogsPage() {
  const [state, setState] = useState<LogsState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;

    async function loadLogs() {
      try {
        const data = await apiFetch<UsageListResponse>("/api/logs?limit=50&offset=0");
        if (!cancelled) {
          setState(data.data.length === 0 ? { status: "empty", data } : { status: "success", data });
        }
      } catch (error) {
        if (!cancelled) {
          setState({
            status: isAuthExpiredError(error) ? "auth-expired" : "error",
            message: error instanceof Error ? error.message : "logs request failed"
          });
        }
      }
    }

    void loadLogs();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Panel title="Request logs" description="Recent gateway request records, bounded to the latest 50 rows.">
      {renderLogs(state)}
    </Panel>
  );
}

function renderLogs(state: LogsState) {
  switch (state.status) {
    case "loading":
      return <LoadingState label="Loading request logs" />;
    case "empty":
      return <EmptyState title="No request logs" description="Request logging has not returned any rows." />;
    case "error":
      return <ErrorState title="Could not load logs" message={state.message} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.message} />;
    case "success":
      return <LogsTable rows={state.data.data} />;
  }
}

function LogsTable({ rows }: { rows: UsageLogRecord[] }) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Request logs" className="min-w-[900px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Request</th>
            <th className="px-4 py-3 font-semibold">Provider</th>
            <th className="px-4 py-3 font-semibold">Model</th>
            <th className="px-4 py-3 font-semibold">Client</th>
            <th className="px-4 py-3 font-semibold">Tokens</th>
            <th className="px-4 py-3 font-semibold">Cost</th>
            <th className="px-4 py-3 font-semibold">Latency</th>
            <th className="px-4 py-3 font-semibold">Status</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {rows.map((row) => (
            <tr key={row.id}>
              <td className="px-4 py-3 font-mono text-xs text-zinc-700">{row.request_id}</td>
              <td className="px-4 py-3 font-medium text-zinc-950">{row.provider}</td>
              <td className="px-4 py-3 text-zinc-600">{row.model}</td>
              <td className="px-4 py-3 text-zinc-600">{row.client_tool ?? "-"}</td>
              <td className="px-4 py-3 text-zinc-600">{row.total_tokens ?? "-"}</td>
              <td className="px-4 py-3 text-zinc-600">{row.cost_usd == null ? "-" : `$${row.cost_usd.toFixed(4)}`}</td>
              <td className="px-4 py-3 text-zinc-600">{row.latency_ms == null ? "-" : `${row.latency_ms}ms`}</td>
              <td className="px-4 py-3">
                <StatusPill tone={statusTone(row)}>{row.status_code ?? "unknown"}</StatusPill>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function statusTone(row: UsageLogRecord): "good" | "warn" | "bad" | "neutral" {
  if (row.error || (row.status_code ?? 0) >= 500) {
    return "bad";
  }
  if ((row.status_code ?? 0) >= 400) {
    return "warn";
  }
  if ((row.status_code ?? 0) >= 200) {
    return "good";
  }
  return "neutral";
}
