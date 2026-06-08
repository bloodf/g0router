import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Button } from "@/components/ui/button";
import { apiFetch } from "@/lib/api/client";
import { toast } from "sonner";
import type { AlertChannel } from "@/lib/types";

export const Route = createFileRoute("/_app/alerts")({
  component: () => (
    <CrudPage<AlertChannel>
      title="Alert Channels"
      description="Configure where gateway notifications are sent."
      icon="notifications"
      endpoint="/api/alert-channels"
      queryKey={["alert-channels"]}
      emptyTitle="No alert channels"
      emptyDescription="Create a channel to receive quota, error, and re-auth alerts."
      fields={[
        { name: "name", label: "Name", required: true },
        {
          name: "channel_type",
          label: "Channel type",
          type: "select",
          options: [
            { label: "Webhook", value: "webhook" },
            { label: "Discord", value: "discord" },
            { label: "Telegram", value: "telegram" },
            { label: "Email", value: "email" },
          ],
          required: true,
        },
        { name: "config", label: "Config (JSON)", type: "textarea", required: true },
        { name: "events", label: "Events (comma-separated)", type: "textarea" },
        { name: "is_active", label: "Active", type: "switch" },
      ]}
      initialValues={(row) => ({
        name: row?.name ?? "",
        channel_type: row?.channel_type ?? "webhook",
        config: row ? JSON.stringify(row.config ?? {}, null, 2) : "{}",
        events: row?.events?.join(", ") ?? "",
        is_active: row?.is_active ?? true,
      })}
      transformBody={(values) => {
        let config = {};
        try {
          config = JSON.parse(values.config || "{}");
        } catch {
          toast.error("Config must be valid JSON");
          throw new Error("Invalid JSON");
        }
        return {
          ...values,
          config,
          events: values.events
            ? String(values.events)
                .split(",")
                .map((s: string) => s.trim())
                .filter(Boolean)
            : [],
        };
      }}
      columns={[
        { header: "Name", accessorKey: "name" },
        {
          header: "Type",
          cell: ({ row }) => (
            <StatusBadge variant="primary">{row.original.channel_type}</StatusBadge>
          ),
        },
        {
          header: "Events",
          cell: ({ row }) => row.original.events?.length ?? 0,
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
      extraActions={(row) => (
        <Button
          variant="ghost"
          size="sm"
          onClick={async () => {
            try {
              await apiFetch(`/api/alert-channels/${row.id}/test`, { method: "POST" });
              toast.success("Test alert sent");
            } catch {
              // apiFetch toasts error
            }
          }}
        >
          Test
        </Button>
      )}
    />
  ),
});
