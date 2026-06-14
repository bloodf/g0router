import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Toggle } from "@/components/ui/toggle";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { AlertChannelFormModal } from "@/components/governance/alert-channel-form-modal";
import { useNotificationStore } from "@/stores/notification";
import type { AlertChannel } from "@/lib/types";

export const Route = createFileRoute("/alerts")({
  component: AlertsPage,
});

// AlertsPage (PAR-UI-130 subset) IS the alert-channels UI: it lists channels from
// GET /api/alert-channels and drives create/edit (AlertChannelFormModal), delete
// (ConfirmModal), the is_active Toggle, and per-channel test (POST
// /api/alert-channels/{id}/test). Variant-HAVE against the mock; no Go
// /api/alert-channels exists yet (§8 ESCALATION-1f).
function AlertsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [channels, setChannels] = React.useState<AlertChannel[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<AlertChannel | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<AlertChannel | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<AlertChannel[]>("/api/alert-channels")
      .then((rows) => {
        setChannels(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setChannels([]);
        setLoading(false);
        pushToast({ message: "Failed to load alert channels" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setActive(channel: AlertChannel, active: boolean) {
    setChannels((prev) =>
      prev.map((c) => (c.id === channel.id ? { ...c, is_active: active } : c))
    );
    try {
      await apiFetch(`/api/alert-channels/${channel.id}`, {
        method: "PUT",
        body: JSON.stringify({ ...channel, is_active: active }),
      });
    } catch {
      pushToast({ message: "Failed to update the channel" });
      load();
    }
  }

  async function testChannel(channel: AlertChannel) {
    try {
      await apiFetch(`/api/alert-channels/${channel.id}/test`, { method: "POST" });
      pushToast({ message: "Test notification sent" });
    } catch {
      pushToast({ message: "Failed to send the test notification" });
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/alert-channels/${deleting.id}`, { method: "DELETE" });
      setChannels((prev) => prev.filter((c) => c.id !== deleting.id));
      pushToast({ message: "Channel deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the channel" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Alerts</h1>
        <Button
          data-testid="alert-channel-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New channel
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : channels.length === 0 ? (
        <p className="text-sm text-muted-foreground">No alert channels yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {channels.map((channel) => (
            <div
              key={channel.id}
              data-testid="alert-channel-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div className="flex flex-col gap-1">
                <p className="text-sm font-medium text-foreground">{channel.name}</p>
                <p className="text-xs text-muted-foreground">
                  {channel.events.join(", ")}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="neutral" size="sm">
                  {channel.channel_type}
                </Badge>
                <Toggle
                  checked={channel.is_active}
                  onCheckedChange={(checked) => setActive(channel, checked)}
                  aria-label={`Toggle ${channel.name}`}
                />
                <Button
                  data-testid="alert-channel-test"
                  variant="ghost"
                  size="sm"
                  onClick={() => testChannel(channel)}
                >
                  Test
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setEditing(channel)}>
                  Edit
                </Button>
                <Button
                  data-testid="alert-channel-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(channel)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <AlertChannelFormModal
        open={creating || editing !== null}
        channel={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete channel"
        message={`Delete "${deleting?.name ?? ""}"? This cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        loading={deleteBusy}
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </div>
  );
}
