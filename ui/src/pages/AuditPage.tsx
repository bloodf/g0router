import { useCallback, useEffect, useState } from "react";
import { ApiError, asyncError, asyncSuccess, listAudit, type AuditListResponse, type AuditLogEntry, type AsyncState } from "../api";
import { EmptyState, ErrorState, LoadingState, Panel } from "../components/Primitives";

export function AuditPage() {
  const [state, setState] = useState<AsyncState<AuditListResponse>>({ status: "loading" });

  const loadAudit = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const result = await listAudit();
      if (result.data.length === 0) {
        setState({ status: "empty", data: result });
      } else {
        setState(asyncSuccess(result));
      }
    } catch (error) {
      setState(asyncError<AuditListResponse>(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadAudit();
  }, [loadAudit]);

  return (
    <Panel title="Audit log" description="Admin mutations recorded by the gateway control plane, newest first.">
      {renderState(state, loadAudit)}
    </Panel>
  );
}

function renderState(state: AsyncState<AuditListResponse>, onRetry: () => void) {
  switch (state.status) {
    case "idle":
    case "loading":
      return <LoadingState label="Loading audit log" />;
    case "empty":
      return <EmptyState title="No audit log entries" description="Admin mutations will appear here once recorded." />;
    case "error":
      return <ErrorState title="Could not load audit log" message={state.error.message} onRetry={onRetry} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.error.message} onRetry={onRetry} />;
    case "success":
      return <AuditTable entries={state.data.data} total={state.data.total} />;
  }
}

function AuditTable({ entries, total }: { entries: AuditLogEntry[]; total: number }) {
  return (
    <div className="space-y-3">
      <p className="text-sm text-zinc-500">{total} total entries</p>
      <div className="overflow-x-auto rounded-md border border-zinc-200">
        <table aria-label="Audit log" className="min-w-[760px] w-full text-left text-sm">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-3 font-semibold">Time</th>
              <th className="px-4 py-3 font-semibold">Actor key</th>
              <th className="px-4 py-3 font-semibold">Action</th>
              <th className="px-4 py-3 font-semibold">Target</th>
              <th className="px-4 py-3 font-semibold">Details</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200">
            {entries.map((entry) => (
              <tr key={entry.id}>
                <td className="px-4 py-3 font-mono text-xs text-zinc-600">{entry.timestamp}</td>
                <td className="px-4 py-3 font-mono text-xs text-zinc-600">{entry.actor_api_key_id || "—"}</td>
                <td className="px-4 py-3 text-zinc-950">{entry.action}</td>
                <td className="px-4 py-3 text-zinc-600">{entry.target}</td>
                <td className="px-4 py-3 text-zinc-600">{entry.details}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "Unknown API error", error);
}
