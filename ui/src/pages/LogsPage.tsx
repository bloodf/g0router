import { useEffect, useMemo, useState } from "react";
import { apiFetch, buildLogsPath, isAuthExpiredError, type LogQuery, type UsageListResponse, type UsageLogRecord } from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

const PAGE_SIZE = 50;

type LogsState =
  | { status: "loading" }
  | { status: "success"; data: UsageListResponse }
  | { status: "empty"; data: UsageListResponse }
  | { status: "error"; message: string }
  | { status: "auth-expired"; message: string };

type Filters = {
  provider: string;
  model: string;
  statusClass: string;
  sourceFormat: string;
  start: string;
  end: string;
};

const emptyFilters: Filters = { provider: "", model: "", statusClass: "", sourceFormat: "", start: "", end: "" };

export function LogsPage() {
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0);
  const [state, setState] = useState<LogsState>({ status: "loading" });

  useEffect(() => {
    const handle = setTimeout(() => setSearch(searchInput), 300);
    return () => clearTimeout(handle);
  }, [searchInput]);

  const query = useMemo<LogQuery>(
    () => ({
      limit: PAGE_SIZE,
      offset,
      provider: filters.provider,
      model: filters.model,
      status_class: filters.statusClass,
      source_format: filters.sourceFormat,
      start: toRFC3339(filters.start),
      end: toRFC3339(filters.end),
      search
    }),
    [filters, search, offset]
  );

  useEffect(() => {
    let cancelled = false;
    setState({ status: "loading" });

    async function loadLogs() {
      try {
        const data = await apiFetch<UsageListResponse>(buildLogsPath(query));
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
  }, [query]);

  function updateFilter<K extends keyof Filters>(key: K, value: Filters[K]) {
    setOffset(0);
    setFilters((current) => ({ ...current, [key]: value }));
  }

  return (
    <Panel title="Request logs" description="Full request log viewer with filtering, search, and pagination.">
      <div className="space-y-4">
        <FilterBar
          filters={filters}
          searchInput={searchInput}
          onFilterChange={updateFilter}
          onSearchChange={(value) => {
            setOffset(0);
            setSearchInput(value);
          }}
        />
        {renderLogs(state)}
        {state.status === "success" || state.status === "empty" ? (
          <Pagination total={state.data.total} count={state.data.data.length} offset={offset} onOffset={setOffset} />
        ) : null}
      </div>
    </Panel>
  );
}

function FilterBar({
  filters,
  searchInput,
  onFilterChange,
  onSearchChange
}: {
  filters: Filters;
  searchInput: string;
  onFilterChange: <K extends keyof Filters>(key: K, value: Filters[K]) => void;
  onSearchChange: (value: string) => void;
}) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      <LabeledInput label="Provider" value={filters.provider} onChange={(value) => onFilterChange("provider", value)} />
      <LabeledInput label="Model" value={filters.model} onChange={(value) => onFilterChange("model", value)} />
      <label className="block text-sm font-medium text-zinc-700">
        Kind
        <select
          aria-label="Kind"
          className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          value={filters.statusClass}
          onChange={(event) => onFilterChange("statusClass", event.target.value)}
        >
          <option value="">All</option>
          <option value="success">success</option>
          <option value="client_error">client_error</option>
          <option value="server_error">server_error</option>
        </select>
      </label>
      <label className="block text-sm font-medium text-zinc-700">
        Source format
        <select
          aria-label="Source format"
          className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          value={filters.sourceFormat}
          onChange={(event) => onFilterChange("sourceFormat", event.target.value)}
        >
          <option value="">All</option>
          <option value="openai">openai</option>
          <option value="anthropic">anthropic</option>
        </select>
      </label>
      <label className="block text-sm font-medium text-zinc-700">
        Start
        <input
          aria-label="Start"
          className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          type="datetime-local"
          value={filters.start}
          onChange={(event) => onFilterChange("start", event.target.value)}
        />
      </label>
      <label className="block text-sm font-medium text-zinc-700">
        End
        <input
          aria-label="End"
          className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          type="datetime-local"
          value={filters.end}
          onChange={(event) => onFilterChange("end", event.target.value)}
        />
      </label>
      <label className="block text-sm font-medium text-zinc-700 xl:col-span-3">
        Search
        <input
          aria-label="Search logs"
          className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          placeholder="Search request id, model, error…"
          type="search"
          value={searchInput}
          onChange={(event) => onSearchChange(event.target.value)}
        />
      </label>
    </div>
  );
}

function LabeledInput({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="block text-sm font-medium text-zinc-700">
      {label}
      <input
        aria-label={label}
        className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}

function Pagination({ total, count, offset, onOffset }: { total: number; count: number; offset: number; onOffset: (value: number) => void }) {
  const from = count === 0 ? 0 : offset + 1;
  const to = offset + count;
  const canPrev = offset > 0;
  const canNext = offset + PAGE_SIZE < total;
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 px-4 py-3 text-sm">
      <span className="text-zinc-600">
        Showing {from}–{to} of {total}
      </span>
      <div className="flex gap-2">
        <button
          className="rounded-md border border-zinc-200 px-3 py-1.5 font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-300"
          disabled={!canPrev}
          type="button"
          onClick={() => onOffset(Math.max(0, offset - PAGE_SIZE))}
        >
          Prev
        </button>
        <button
          className="rounded-md border border-zinc-200 px-3 py-1.5 font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-300"
          disabled={!canNext}
          type="button"
          onClick={() => onOffset(offset + PAGE_SIZE)}
        >
          Next
        </button>
      </div>
    </div>
  );
}

function renderLogs(state: LogsState) {
  switch (state.status) {
    case "loading":
      return <LoadingState label="Loading request logs" />;
    case "empty":
      return <EmptyState title="No logs match" description="No request logs match the current filters." />;
    case "error":
      return <ErrorState title="Could not load logs" message={state.message} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.message} />;
    case "success":
      return <LogsTable rows={state.data.data} />;
  }
}

function LogsTable({ rows }: { rows: UsageLogRecord[] }) {
  const [expanded, setExpanded] = useState<number | null>(null);
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Request logs" className="min-w-[900px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Timestamp</th>
            <th className="px-4 py-3 font-semibold">Provider</th>
            <th className="px-4 py-3 font-semibold">Model</th>
            <th className="px-4 py-3 font-semibold">Status</th>
            <th className="px-4 py-3 font-semibold">Latency</th>
            <th className="px-4 py-3 font-semibold">Cost</th>
            <th className="px-4 py-3 font-semibold">Client</th>
            <th className="px-4 py-3 font-semibold">Combo</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {rows.map((row) => (
            <LogRow key={row.id} row={row} expanded={expanded === row.id} onToggle={() => setExpanded(expanded === row.id ? null : row.id)} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function LogRow({ row, expanded, onToggle }: { row: UsageLogRecord; expanded: boolean; onToggle: () => void }) {
  return (
    <>
      <tr
        aria-expanded={expanded}
        className="cursor-pointer hover:bg-zinc-50"
        onClick={onToggle}
      >
        <td className="px-4 py-3 font-mono text-xs text-zinc-700">{row.timestamp}</td>
        <td className="px-4 py-3 font-medium text-zinc-950">{row.provider}</td>
        <td className="px-4 py-3 text-zinc-600">{row.model}</td>
        <td className="px-4 py-3">
          <StatusPill tone={statusTone(row)}>{row.status_code ?? "unknown"}</StatusPill>
        </td>
        <td className="px-4 py-3 text-zinc-600">{row.latency_ms == null ? "-" : `${row.latency_ms}ms`}</td>
        <td className="px-4 py-3 text-zinc-600">{row.cost_usd == null ? "-" : `$${row.cost_usd.toFixed(4)}`}</td>
        <td className="px-4 py-3 text-zinc-600">{row.client_tool ?? "-"}</td>
        <td className="px-4 py-3 text-zinc-600">{row.combo_name ?? "-"}</td>
      </tr>
      {expanded ? (
        <tr className="bg-zinc-50">
          <td className="px-4 py-3 text-xs text-zinc-600" colSpan={8}>
            <dl className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
              <Detail label="Request ID" value={row.request_id} mono />
              <Detail label="API key" value={row.api_key_id ?? "-"} mono />
              <Detail label="Source / target" value={`${row.source_format ?? "-"} → ${row.target_format ?? "-"}`} />
              <Detail label="Input tokens" value={String(row.input_tokens ?? "-")} />
              <Detail label="Output tokens" value={String(row.output_tokens ?? "-")} />
              <Detail label="Total tokens" value={String(row.total_tokens ?? "-")} />
              <Detail label="RTK bytes saved" value={String(row.rtk_bytes_saved ?? "-")} />
              {row.error ? <Detail label="Error" value={row.error} /> : null}
            </dl>
          </td>
        </tr>
      ) : null}
    </>
  );
}

function Detail({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="font-semibold uppercase text-zinc-500">{label}</dt>
      <dd className={mono ? "font-mono text-zinc-700" : "text-zinc-700"}>{value}</dd>
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

function toRFC3339(local: string): string | undefined {
  if (!local) {
    return undefined;
  }
  const parsed = new Date(local);
  if (Number.isNaN(parsed.getTime())) {
    return undefined;
  }
  return parsed.toISOString();
}
