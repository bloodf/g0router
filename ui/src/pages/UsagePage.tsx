import { useEffect, useState } from "react";
import { isAuthExpiredError, listAPIKeys, listLogs, listUsage } from "../api";
import type { APIKeyResponse, UsageLogRecord, UsageQuery } from "../api";
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

type UsageFilters = {
  apiKeyId: string;
  authType: string;
};

const emptyFilters: UsageFilters = { apiKeyId: "", authType: "" };

export function UsagePage() {
  const [state, setState] = useState<UsageState>({ status: "loading" });
  const [filters, setFilters] = useState<UsageFilters>(emptyFilters);
  const [apiKeys, setApiKeys] = useState<APIKeyResponse[]>([]);

  useEffect(() => {
    let cancelled = false;
    listAPIKeys()
      .then((keys) => {
        if (!cancelled) {
          setApiKeys(keys);
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setState({ status: "loading" });

    async function loadUsage() {
      const query: UsageQuery = {};
      if (filters.apiKeyId) {
        query.api_key_id = filters.apiKeyId;
      }
      if (filters.authType) {
        query.auth_type = filters.authType;
      }
      try {
        const [usage, logs] = await Promise.all([listUsage(query), listLogs()]);
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
  }, [filters]);

  return (
    <div className="space-y-6">
      {state.status === "success" ? <UsageMetrics data={state.data} /> : null}

      <Panel
        title="Usage history"
        description="Per-request usage history with key and account attribution. For raw request logs, see the Logs page."
      >
        <UsageFiltersBar apiKeys={apiKeys} filters={filters} onChange={setFilters} />
        {renderUsageContent(state)}
      </Panel>
    </div>
  );
}

function UsageFiltersBar({
  apiKeys,
  filters,
  onChange
}: {
  apiKeys: APIKeyResponse[];
  filters: UsageFilters;
  onChange: (filters: UsageFilters) => void;
}) {
  return (
    <div className="mb-4 flex flex-wrap items-end gap-3">
      <label className="text-sm font-medium text-zinc-700">
        API key
        <select
          aria-label="Filter by API key"
          className="mt-1 block w-56 rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          value={filters.apiKeyId}
          onChange={(event) => onChange({ ...filters, apiKeyId: event.target.value })}
        >
          <option value="">All API keys</option>
          {apiKeys.map((key) => (
            <option key={key.ID} value={key.ID}>
              {key.Name}
            </option>
          ))}
        </select>
      </label>
      <label className="text-sm font-medium text-zinc-700">
        Auth type
        <select
          aria-label="Filter by auth type"
          className="mt-1 block w-40 rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          value={filters.authType}
          onChange={(event) => onChange({ ...filters, authType: event.target.value })}
        >
          <option value="">All auth types</option>
          <option value="oauth">OAuth</option>
          <option value="api_key">API key</option>
        </select>
      </label>
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
      <div className="overflow-x-auto rounded-md border border-zinc-200">
        <table aria-label="Usage rows" className="min-w-[820px] w-full text-left text-sm">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-3 font-semibold">Request</th>
              <th className="px-4 py-3 font-semibold">Provider</th>
              <th className="px-4 py-3 font-semibold">Model</th>
              <th className="px-4 py-3 font-semibold">Key</th>
              <th className="px-4 py-3 font-semibold">Account / OAuth</th>
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
                <td className="px-4 py-3 text-zinc-600">{formatKey(record)}</td>
                <td className="px-4 py-3 text-zinc-600">
                  <div className="flex flex-col gap-1">
                    <span>{formatAccount(record)}</span>
                    <StatusPill tone={authTone(record)}>{authLabel(record)}</StatusPill>
                  </div>
                </td>
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
        <div className="overflow-x-auto rounded-md border border-zinc-200">
          <table aria-label="Request logs" className="min-w-[720px] w-full text-left text-sm">
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

function formatKey(record: UsageLogRecord) {
  if (record.api_key_name) {
    return record.api_key_name;
  }
  if (record.api_key_id) {
    return record.api_key_id.slice(0, 8);
  }
  return "-";
}

function formatAccount(record: UsageLogRecord) {
  const provider = record.connection_provider ?? "";
  const email = record.account_email ?? "";
  if (provider && email) {
    return `${provider} · ${email}`;
  }
  return provider || email || "-";
}

function authLabel(record: UsageLogRecord) {
  return record.auth_type === "oauth" ? "oauth" : "api_key";
}

function authTone(record: UsageLogRecord): "good" | "neutral" {
  return record.auth_type === "oauth" ? "good" : "neutral";
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
