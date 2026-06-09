import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { apiFetch } from "@/lib/api/client";
import { toast } from "sonner";
import type { ProxyPool } from "@/lib/types";

export const Route = createFileRoute("/_app/proxy-pools")({
  component: () => <ProxyPoolsPage />,
});

function ProxyPoolsPage() {
  const qc = useQueryClient();
  const [batchOpen, setBatchOpen] = useState(false);
  const [batchText, setBatchText] = useState("");

  const batch = useMutation({
    mutationFn: (lines: string[]) =>
      apiFetch<{ created: ProxyPool[]; errors: { line: string; error: string }[] }>(
        "/api/proxy-pools/batch",
        { method: "POST", body: { lines } },
      ),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["proxy-pools"] });
      toast.success(`Imported ${res.created.length} proxies`);
      if (res.errors.length) {
        toast.error(`${res.errors.length} lines failed`);
      }
      setBatchOpen(false);
      setBatchText("");
    },
    onError: (e: any) => toast.error(e?.message || "Batch import failed"),
  });

  return (
    <>
      <CrudPage<ProxyPool>
        title="Proxy Pools"
        description="Outbound proxies for provider requests."
        icon="lan"
        endpoint="/api/proxy-pools"
        queryKey={["proxy-pools"]}
        emptyTitle="No proxy pools"
        emptyDescription="Add HTTP, HTTPS, or SOCKS5 proxies to route provider traffic."
        fields={[
          { name: "name", label: "Name", required: true },
          {
            name: "protocol",
            label: "Protocol",
            type: "select",
            options: [
              { label: "HTTP", value: "http" },
              { label: "HTTPS", value: "https" },
              { label: "SOCKS5", value: "socks5" },
            ],
            required: true,
          },
          { name: "host", label: "Host", required: true },
          { name: "port", label: "Port", type: "number", required: true },
          { name: "username", label: "Username" },
          { name: "password", label: "Password", placeholder: "Leave blank to keep current" },
        ]}
        initialValues={(row) => ({
          name: row?.name ?? "",
          protocol: row?.protocol ?? "http",
          host: row?.host ?? "",
          port: row?.port ?? "",
          username: row?.username ?? "",
          password: "",
        })}
        transformBody={(values) => ({
          ...values,
          port: Number(values.port),
          password: values.password || undefined,
        })}
        columns={[
          { header: "Name", accessorKey: "name" },
          { header: "Protocol", accessorKey: "protocol" },
          {
            header: "Host:Port",
            cell: ({ row }) => `${row.original.host}:${row.original.port}`,
          },
          {
            header: "Username",
            cell: ({ row }) => row.original.username || "—",
          },
          {
            header: "Status",
            cell: ({ row }) => (
              <StatusBadge variant={row.original.is_active ? "success" : "muted"} dot>
                {row.original.is_active ? "active" : "inactive"}
              </StatusBadge>
            ),
          },
          {
            header: "Last check",
            cell: ({ row }) => row.original.last_check_status || "—",
          },
        ]}
        extraToolbar={
          <Button variant="outline" onClick={() => setBatchOpen(true)}>
            Batch import
          </Button>
        }
        extraActions={(row) => (
          <Button
            variant="ghost"
            size="sm"
            onClick={async () => {
              try {
                const res = await apiFetch<{ ok: boolean; latency_ms: number; error: string }>(
                  `/api/proxy-pools/${row.id}/test`,
                  { method: "POST" },
                );
                if (res.ok) {
                  toast.success(`Proxy OK — ${res.latency_ms}ms`);
                } else {
                  toast.error(res.error || "Proxy test failed");
                }
              } catch {
                // apiFetch toasts error
              }
            }}
          >
            Test
          </Button>
        )}
      />

      <Dialog open={batchOpen} onOpenChange={setBatchOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Batch import proxies</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <textarea
              className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none min-h-[160px]"
              value={batchText}
              onChange={(e) => setBatchText(e.target.value)}
              placeholder={`http://user:pass@host:port\nhost:port\nsocks5://host:1080`}
              aria-label="Proxy list"
            />
            <p className="text-xs text-text-muted">
              One proxy per line. Supported formats: host:port, protocol://host:port, and
              protocol://user:pass@host:port.
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBatchOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() =>
                batch.mutate(
                  batchText
                    .split("\n")
                    .map((l) => l.trim())
                    .filter(Boolean),
                )
              }
              disabled={batch.isPending || !batchText.trim()}
            >
              {batch.isPending ? "Importing…" : "Import"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
