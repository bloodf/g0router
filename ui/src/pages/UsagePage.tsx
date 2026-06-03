import { useEffect, useState } from "react";
import { isAuthExpiredError, listLogs, listUsage } from "../api";
import type { UsageLogRecord } from "../api";
import { EmptyState, ErrorState, LoadingState, MetricCard, Panel, StatusPill } from "../components/Primitives";

type UsageData = {
  logs: UsageLogRecord[];
  usage: UsageLogRecord[];
};

type UsageState =
  | { status: "loading" }
  | { status: "success"; data: UsageData }
  | { status: "empty"; data: UsageData }
  | { status: "error"; message: string }
  | { status: "auth-expired"; message: string };

export function UsagePage() {
  const [state, setState] = useState<UsageState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;

    async function loadUsage() {
      try {
        const [usage, logs] = await Promise.all([listUsage(), listLogs()]);
        if (cancelled) {
          return;
        }

        const data = { usage: usage.data, logs: logs.data };
        setState(data.usage.length === 0 && data.logs.length === 0 ? { status: "empty", data } : { status: "success", data });
      } catch (error) {
        if (cancelled) {
          return;
        }
        setState({
          status: isAuthExpiredError(error) ? "auth-expired" : "error",
          message: error instanceof Error ? error.message : "usage request failed"
        });
      }
    }

    void loadUsage();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="space-y-6">
      {state.status === "success" ? <UsageMetrics data={state.data} /> : null}

      <Panel title="Usage analytics" description="Request, token, cost, and log rows returned by the management API.">
        {renderUsageContent(state)}
      </Panel>
    </div>
  );
}

function UsageMetrics({ data }: { data: UsageData }) {
  const allRows = [...data.usage, ...data.logs];
  const totalTokens = allRows.reduce((sum, record) => sum + (record.total_tokens ?? 0), 0);
  const totalCost = allRows.reduce((sum, record) => sum + (record.cost_usd ?? 0), 0);
  const failedRows = allRows.filter(isFailedRecord);

  return (
    <div className="grid gap-4 md:grid-cols-3">
      <MetricCard label="Usage rows" value={formatInteger(data.usage.length)} detail={`${formatInteger(data.logs.length)} log rows loaded`} tone="sky" />
      <MetricCard label="Tokens tracked" value={formatCompactNumber(totalTokens)} detail={`${formatUSD(totalCost)} from returned rows`} tone="emerald" />
      <MetricCard
        label="Failed rows"
        value={formatInteger(failedRows.length)}
        detail={`${formatInteger(allRows.filter(isStreamingRecord).length)} streaming rows`}
        tone={failedRows.length > 0 ? "amber" : "emerald"}
      />
    </div>
  );
}

function renderUsageContent(state: UsageState) {
  switch (state.status) {
    case "loading":
      return <LoadingState label="Loading usage data" />;
    case "empty":
      return <EmptyState title="No usage or logs yet" description="The usage and log APIs returned empty lists." />;
    case "error":
      return <ErrorState title="Usage data unavailable" message={state.message} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.message} />;
    case "success":
      return <UsageTables data={state.data} />;
  }
}

function UsageTables({ data }: { data: UsageData }) {
  return (
    <div className="space-y-6">
      <div className="overflow-hidden rounded-md border border-zinc-200">
        <table className="w-full text-left text-sm">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-3 font-semibold">Request</th>
              <th className="px-4 py-3 font-semibold">Provider</th>
              <th className="px-4 py-3 font-semibold">Model</th>
              <th className="px-4 py-3 font-semibold">Tokens</th>
              <th className="px-4 py-3 font-semibold">Cost</th>
              <th className="px-4 py-3 font-semibold">Status</th>
              <th className="px-4 py-3 font-semibold">Detail</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200">
            {data.usage.map((record) => (
              <tr key={record.id}>
                <td className="px-4 py-3 font-mono text-xs text-zinc-700">{record.request_id}</td>
                <td className="px-4 py-3 font-medium text-zinc-950">{record.provider}</td>
                <td className="px-4 py-3 text-zinc-600">{record.model}</td>
                <td className="px-4 py-3 text-zinc-600">{formatNullableInteger(record.total_tokens)}</td>
                <td className="px-4 py-3 text-zinc-600">{formatNullableUSD(record.cost_usd)}</td>
                <td className="px-4 py-3">
                  <StatusPill tone={statusTone(record)}>{statusLabel(record)}</StatusPill>
                </td>
                <td className="px-4 py-3 text-zinc-600">{record.error ?? record.client_tool ?? "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div>
        <h4 className="mb-3 text-sm font-semibold text-zinc-700">Request logs</h4>
        <div className="overflow-hidden rounded-md border border-zinc-200">
          <table className="w-full text-left text-sm">
            <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-4 py-3 font-semibold">Request</th>
                <th className="px-4 py-3 font-semibold">Client</th>
                <th className="px-4 py-3 font-semibold">Format</th>
                <th className="px-4 py-3 font-semibold">Latency</th>
                <th className="px-4 py-3 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {data.logs.map((record) => (
                <tr key={record.id}>
                  <td className="px-4 py-3 font-mono text-xs text-zinc-700">{record.request_id}</td>
                  <td className="px-4 py-3 text-zinc-600">{record.client_tool ?? "-"}</td>
                  <td className="px-4 py-3 text-zinc-600">
                    <div className="flex items-center gap-2">
                      <span>{formatFlow(record)}</span>
                      {isStreamingRecord(record) ? <StatusPill tone="neutral">streaming</StatusPill> : <StatusPill tone="neutral">log</StatusPill>}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-zinc-600">{record.latency_ms == null ? "-" : `${record.latency_ms}ms`}</td>
                  <td className="px-4 py-3">
                    <StatusPill tone={statusTone(record)}>{statusLabel(record)}</StatusPill>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function isFailedRecord(record: UsageLogRecord) {
  return Boolean(record.error) || (record.status_code ?? 0) >= 400;
}

function isStreamingRecord(record: UsageLogRecord) {
  const source = record.source_format?.toLowerCase() ?? "";
  const target = record.target_format?.toLowerCase() ?? "";
  return source.includes("stream") || source.includes("sse") || target.includes("stream") || target.includes("sse");
}

function statusTone(record: UsageLogRecord): "good" | "warn" | "bad" | "neutral" {
  if (record.error || (record.status_code ?? 0) >= 500) {
    return "bad";
  }
  if ((record.status_code ?? 0) >= 400) {
    return "warn";
  }
  if ((record.status_code ?? 0) >= 200) {
    return "good";
  }
  return "neutral";
}

function statusLabel(record: UsageLogRecord) {
  return record.status_code == null ? "unknown" : String(record.status_code);
}

function formatFlow(record: UsageLogRecord) {
  return `${record.source_format ?? "-"} -> ${record.target_format ?? "-"}`;
}

function formatNullableInteger(value?: number | null) {
  return value == null ? "-" : value.toLocaleString();
}

function formatInteger(value: number) {
  return value.toLocaleString();
}

function formatCompactNumber(value: number) {
  if (Math.abs(value) >= 1000) {
    return `${(value / 1000).toFixed(1)}k`;
  }
  return formatInteger(value);
}

function formatNullableUSD(value?: number | null) {
  return value == null ? "-" : formatUSD(value);
}

function formatUSD(value: number) {
  return `$${value.toFixed(3)}`;
}
