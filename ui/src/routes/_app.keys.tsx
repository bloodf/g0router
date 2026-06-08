import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import { CopyButton } from "@/components/common/CopyButton";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/common/Icon";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { apiFetch } from "@/lib/api/client";
import { format } from "date-fns";
import type { ApiKey } from "@/lib/types";

const SCOPE_OPTIONS = [
  { label: "gpt-*", value: "gpt-*" },
  { label: "claude-*", value: "claude-*" },
  { label: "gemini-*", value: "gemini-*" },
  { label: "llama-*", value: "llama-*" },
  { label: "mistral-*", value: "mistral-*" },
  { label: "deepseek-*", value: "deepseek-*" },
];

function toUnixSeconds(dateStr: string): number | null {
  if (!dateStr) return null;
  const ms = new Date(dateStr).getTime();
  if (Number.isNaN(ms)) return null;
  return Math.floor(ms / 1000);
}

function fromUnixSeconds(value: unknown): string {
  if (value === null || value === undefined || value === "") return "";
  const ts = typeof value === "number" ? value : Number(value);
  if (Number.isNaN(ts)) return "";
  // Backend stores seconds; convert to milliseconds for Date.
  const ms = ts < 1e12 ? ts * 1000 : ts;
  const d = new Date(ms);
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

function formatExpiresAt(value: unknown): string {
  const dateStr = fromUnixSeconds(value);
  if (!dateStr) return "Never";
  return format(new Date(dateStr), "MMM d, yyyy");
}

export const Route = createFileRoute("/_app/keys")({
  component: KeysPage,
});

function KeysPage() {
  const qc = useQueryClient();
  const [regeneratedKey, setRegeneratedKey] = useState<ApiKey | null>(null);

  const regenerate = useMutation({
    mutationFn: (id: string) =>
      apiFetch<ApiKey>(`/api/keys/${id}/regenerate`, { method: "POST" }),
    onSuccess: (k) => {
      qc.invalidateQueries({ queryKey: ["keys"] });
      setRegeneratedKey(k);
    },
  });

  return (
    <>
      <CrudPage<ApiKey>
        title="API Keys"
        description="OpenAI-compatible keys for /v1 endpoints."
        icon="key"
        endpoint="/api/keys"
        queryKey={["keys"]}
        emptyTitle="No API keys yet"
        emptyDescription="Generate an OpenAI-compatible key to call the /v1 endpoints from your apps."
        fields={[
          { name: "name", label: "Name", required: true },
          {
            name: "scopes",
            label: "Scopes",
            type: "multiselect",
            options: SCOPE_OPTIONS,
          },
          { name: "expires_at", label: "Expires at", type: "date" },
          { name: "rpm_limit", label: "RPM limit", type: "number" },
          { name: "tpm_limit", label: "TPM limit", type: "number" },
          {
            name: "daily_spend_cap",
            label: "Daily spend cap ($)",
            type: "number",
          },
        ]}
        initialValues={(row) => ({
          name: row?.name ?? "",
          scopes: row?.scopes ?? [],
          expires_at: row ? fromUnixSeconds(row.expires_at) : "",
          rpm_limit: row?.rpm_limit ?? "",
          tpm_limit: row?.tpm_limit ?? "",
          daily_spend_cap: row?.daily_spend_cap ?? "",
        })}
        transformBody={(values) => ({
          ...values,
          expires_at: toUnixSeconds(values.expires_at),
        })}
        extraActions={(row) => (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => regenerate.mutate(row.id)}
            disabled={regenerate.isPending}
            title="Regenerate key — invalidates the old value"
          >
            <Icon
              name={regenerate.isPending ? "hourglass_empty" : "autorenew"}
              size={14}
              className={
                "mr-1 " + (regenerate.isPending ? "animate-spin" : "")
              }
            />
            {regenerate.isPending ? "Regenerating…" : "Regenerate"}
          </Button>
        )}
        columns={[
          { header: "Name", accessorKey: "name" },
          {
            header: "Prefix",
            cell: ({ row }) => (
              <div className="flex items-center gap-1">
                <code className="text-xs bg-surface-2 px-1.5 py-0.5 rounded">
                  {row.original.prefix}…
                </code>
                {row.original.full_key && (
                  <CopyButton value={row.original.full_key} />
                )}
              </div>
            ),
          },
          {
            header: "Scopes",
            cell: ({ row }) =>
              row.original.scopes?.length ? (
                <span className="text-xs text-text-muted">
                  {row.original.scopes.join(", ")}
                </span>
              ) : (
                "—"
              ),
          },
          {
            header: "RPM",
            accessorKey: "rpm_limit",
            cell: ({ row }) => row.original.rpm_limit ?? "—",
          },
          {
            header: "Cap",
            cell: ({ row }) =>
              row.original.daily_spend_cap
                ? `$${row.original.daily_spend_cap}/d`
                : "—",
          },
          {
            header: "Expires",
            cell: ({ row }) => formatExpiresAt(row.original.expires_at),
          },
          {
            header: "Status",
            cell: ({ row }) => (
              <StatusBadge variant={row.original.is_active ? "success" : "muted"} dot>
                {row.original.is_active ? "active" : "inactive"}
              </StatusBadge>
            ),
          },
        ]}
      />

      <Dialog open={!!regeneratedKey} onOpenChange={() => setRegeneratedKey(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>API key regenerated</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-text-muted">
            Copy the full key now. It will not be shown again.
          </p>
          <div className="flex items-center gap-2 bg-surface-2 border border-border rounded-xl px-4 py-3 font-mono text-sm">
            <span className="flex-1 truncate select-all">
              {regeneratedKey?.full_key}
            </span>
            <CopyButton value={regeneratedKey?.full_key ?? ""} variant="outline" />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
