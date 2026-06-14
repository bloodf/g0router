import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Select } from "@/components/ui/select";
import { CardSkeleton } from "@/components/ui/skeleton";
import { useNotificationStore } from "@/stores/notification";
import type { AuditLog } from "@/lib/types";

export const Route = createFileRoute("/audit")({
  component: AuditPage,
});

interface AuditResponse {
  items: AuditLog[];
  total: number;
}

// AuditPage (PAR-UI-130 subset) is a read-only audit-log viewer over
// GET /api/audit?limit= (the mock returns the paginated {items,total} shape).
// Variant-HAVE against the mock; no Go /api/audit exists yet (§8 ESCALATION-1b).
function AuditPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [logs, setLogs] = React.useState<AuditLog[]>([]);
  const [total, setTotal] = React.useState(0);
  const [limit, setLimit] = React.useState("25");
  const [loading, setLoading] = React.useState(true);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<AuditResponse>("/api/audit")
      .then((data) => {
        setLogs(data?.items ?? []);
        setTotal(data?.total ?? 0);
        setLoading(false);
      })
      .catch(() => {
        setLogs([]);
        setLoading(false);
        pushToast({ message: "Failed to load the audit log" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  // The limit is applied client-side over the loaded events.
  const visibleLogs = React.useMemo(
    () => logs.slice(0, Number(limit) || logs.length),
    [logs, limit]
  );

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Audit</h1>
        <div className="w-32">
          <Select
            id="audit-limit"
            data-testid="audit-limit"
            aria-label="Rows per page"
            value={limit}
            onChange={(event) => setLimit(event.target.value)}
            options={[
              { value: "25", label: "25 rows" },
              { value: "50", label: "50 rows" },
              { value: "100", label: "100 rows" },
            ]}
          />
        </div>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : visibleLogs.length === 0 ? (
        <p className="text-sm text-muted-foreground">No audit events yet.</p>
      ) : (
        <div className="flex flex-col gap-3">
          <div className="overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-border text-xs text-muted-foreground">
                <tr>
                  <th className="px-4 py-2 font-medium">Timestamp</th>
                  <th className="px-4 py-2 font-medium">Actor</th>
                  <th className="px-4 py-2 font-medium">Action</th>
                  <th className="px-4 py-2 font-medium">Target</th>
                  <th className="px-4 py-2 font-medium">Details</th>
                </tr>
              </thead>
              <tbody>
                {visibleLogs.map((log) => (
                  <tr
                    key={log.id}
                    data-testid="audit-row"
                    className="border-b border-border last:border-0"
                  >
                    <td className="px-4 py-2 text-muted-foreground">
                      {new Date(log.timestamp).toLocaleString()}
                    </td>
                    <td className="px-4 py-2 text-foreground">{log.actor}</td>
                    <td className="px-4 py-2 text-foreground">{log.action}</td>
                    <td className="px-4 py-2 text-muted-foreground">{log.target}</td>
                    <td className="px-4 py-2 text-muted-foreground">
                      {log.details ?? "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-xs text-muted-foreground">
            Showing {visibleLogs.length} of {total} events.
          </p>
        </div>
      )}
    </div>
  );
}
