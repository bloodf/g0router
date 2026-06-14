import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Pagination } from "@/components/ui/pagination";
import { Badge } from "@/components/ui/badge";
import { CardSkeleton } from "@/components/ui/skeleton";
import type { UsageLog } from "@/lib/types";

// RequestDetailsTab renders paginated request details from
// GET /api/usage/request-details (internal/admin/usage.go:145), response shape
// {data:{data:<rows>, pagination:{page,page_size,total,total_pages}}}.
interface Pagination_ {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}
interface DetailsResponse {
  data: UsageLog[];
  pagination: Pagination_;
}

export function RequestDetailsTab() {
  const [page, setPage] = React.useState(1);
  const [rows, setRows] = React.useState<UsageLog[]>([]);
  const [totalPages, setTotalPages] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [provider, setProvider] = React.useState("");
  const [model, setModel] = React.useState("");
  const [status, setStatus] = React.useState("");

  React.useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const params = new URLSearchParams({ page: String(page), pageSize: "20" });
    if (provider) params.set("provider", provider);
    if (model) params.set("model", model);
    if (status) params.set("status", status);
    apiFetch<DetailsResponse>(`/api/usage/request-details?${params.toString()}`)
      .then((res) => {
        if (cancelled) return;
        setRows(res?.data ?? []);
        setTotalPages(res?.pagination?.total_pages ?? 1);
        setLoading(false);
      })
      .catch(() => {
        if (cancelled) return;
        setRows([]);
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [page, provider, model, status]);

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <Input id="details-provider" label="Provider" value={provider} onChange={(e) => { setProvider(e.target.value); setPage(1); }} />
        <Input id="details-model" label="Model" value={model} onChange={(e) => { setModel(e.target.value); setPage(1); }} />
        <Input id="details-status" label="Status" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1); }} />
      </div>

      {loading ? (
        <CardSkeleton />
      ) : (
        <div className="overflow-x-auto rounded-xl border border-border bg-card">
          <table className="w-full text-sm" data-testid="request-details-table">
            <thead>
              <tr className="border-b border-border text-left text-muted-foreground">
                <th className="px-4 py-2 font-medium">Time</th>
                <th className="px-4 py-2 font-medium">Provider</th>
                <th className="px-4 py-2 font-medium">Model</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Tokens</th>
                <th className="px-4 py-2 font-medium">Cost</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id} className="border-b border-border/50">
                  <td className="whitespace-nowrap px-4 py-2 text-muted-foreground">{row.timestamp}</td>
                  <td className="px-4 py-2">{row.provider}</td>
                  <td className="px-4 py-2 text-foreground">{row.model}</td>
                  <td className="px-4 py-2">
                    <Badge variant={row.status === "success" ? "success" : "error"}>{row.status}</Badge>
                  </td>
                  <td className="px-4 py-2">{row.total_tokens}</td>
                  <td className="px-4 py-2">${row.cost_usd.toFixed(4)}</td>
                </tr>
              ))}
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-6 text-center text-muted-foreground">
                    No matching requests.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      )}

      <div className="flex justify-end">
        <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
      </div>
    </div>
  );
}
