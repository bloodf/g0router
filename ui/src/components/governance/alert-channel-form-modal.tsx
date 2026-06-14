import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Toggle } from "@/components/ui/toggle";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { AlertChannel } from "@/lib/types";

export interface AlertChannelFormModalProps {
  open: boolean;
  channel: AlertChannel | null;
  onClose: () => void;
  onSaved?: () => void;
}

// AlertChannelFormModal creates/edits an alert channel via POST /api/alert-channels
// (new) or PUT /api/alert-channels/{id} (edit). Variant-HAVE against the mock;
// no Go /api/alert-channels exists yet (§8 ESCALATION-1f).
function AlertChannelFormModal({
  open,
  channel,
  onClose,
  onSaved,
}: AlertChannelFormModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [channelType, setChannelType] = React.useState("webhook");
  const [url, setUrl] = React.useState("");
  const [eventsText, setEventsText] = React.useState("");
  const [isActive, setIsActive] = React.useState(true);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (channel) {
      setName(channel.name);
      setChannelType(channel.channel_type);
      const cfg = channel.config ?? {};
      setUrl(String(cfg.url ?? cfg.webhook_url ?? ""));
      setEventsText(channel.events.join(", "));
      setIsActive(channel.is_active);
    } else {
      setName("");
      setChannelType("webhook");
      setUrl("");
      setEventsText("");
      setIsActive(true);
    }
  }, [channel]);

  async function save() {
    setBusy(true);
    const payload = {
      name,
      channel_type: channelType,
      config: channelType === "discord" ? { webhook_url: url } : { url },
      events: eventsText
        .split(",")
        .map((entry) => entry.trim())
        .filter(Boolean),
      is_active: isActive,
    };
    try {
      if (channel) {
        await apiFetch(`/api/alert-channels/${channel.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/alert-channels", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: channel ? "Channel updated" : "Channel created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the channel" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={channel ? "Edit channel" : "New channel"}
    >
      <div className="flex flex-col gap-4">
        <Input
          id="alert-channel-name"
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <Select
          id="alert-channel-type"
          label="Channel type"
          value={channelType}
          onChange={(event) => setChannelType(event.target.value)}
          options={[
            { value: "webhook", label: "Webhook" },
            { value: "discord", label: "Discord" },
            { value: "slack", label: "Slack" },
          ]}
        />
        <Input
          id="alert-channel-url"
          label="URL / webhook URL"
          value={url}
          onChange={(event) => setUrl(event.target.value)}
        />
        <Input
          id="alert-channel-events"
          label="Events (comma-separated)"
          value={eventsText}
          onChange={(event) => setEventsText(event.target.value)}
        />
        <label className="flex items-center justify-between text-sm text-foreground">
          Active
          <Toggle checked={isActive} onCheckedChange={setIsActive} aria-label="Active" />
        </label>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="alert-channel-save"
            variant="primary"
            loading={busy}
            onClick={save}
          >
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { AlertChannelFormModal };
