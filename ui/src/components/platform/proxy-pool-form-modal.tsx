import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Toggle } from "@/components/ui/toggle";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import { toProxyPoolPayload, type ProxyPoolForm } from "@/lib/proxy-pool-form";
import type { ProxyPool } from "@/lib/types";

export interface ProxyPoolFormModalProps {
  open: boolean;
  pool: ProxyPool | null;
  onClose: () => void;
  onSaved?: () => void;
}

const PROTOCOL_OPTIONS = [
  { value: "http", label: "HTTP" },
  { value: "https", label: "HTTPS" },
  { value: "socks5", label: "SOCKS5" },
];

// ProxyPoolFormModal creates/edits a proxy pool via POST /api/proxy-pools (new)
// or PUT /api/proxy-pools/{id} (edit), using the pure toProxyPoolPayload helper.
// PARTIAL against the registered mock; no Go backend exists yet (§1.4/§8 ESC-1b).
function ProxyPoolFormModal({ open, pool, onClose, onSaved }: ProxyPoolFormModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [form, setForm] = React.useState<ProxyPoolForm>({
    name: "",
    protocol: "https",
    host: "",
    port: "8080",
    username: "",
    is_active: true,
  });
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (pool) {
      setForm({
        name: pool.name,
        protocol: pool.protocol,
        host: pool.host,
        port: String(pool.port),
        username: pool.username,
        is_active: pool.is_active,
      });
    } else {
      setForm({
        name: "",
        protocol: "https",
        host: "",
        port: "8080",
        username: "",
        is_active: true,
      });
    }
  }, [pool]);

  function patch(partial: Partial<ProxyPoolForm>) {
    setForm((prev) => ({ ...prev, ...partial }));
  }

  async function save() {
    setBusy(true);
    const payload = toProxyPoolPayload(form);
    try {
      if (pool) {
        await apiFetch(`/api/proxy-pools/${pool.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/proxy-pools", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: pool ? "Pool updated" : "Pool created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the pool" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={pool ? "Edit pool" : "New pool"}>
      <div className="flex flex-col gap-4">
        <Input
          id="proxy-pool-name"
          label="Name"
          value={form.name}
          onChange={(event) => patch({ name: event.target.value })}
        />
        <Select
          id="proxy-pool-protocol"
          label="Protocol"
          options={PROTOCOL_OPTIONS}
          value={form.protocol}
          onChange={(event) => patch({ protocol: event.target.value })}
        />
        <Input
          id="proxy-pool-host"
          label="Host"
          value={form.host}
          onChange={(event) => patch({ host: event.target.value })}
        />
        <Input
          id="proxy-pool-port"
          label="Port"
          type="number"
          value={form.port}
          onChange={(event) => patch({ port: event.target.value })}
        />
        <Input
          id="proxy-pool-username"
          label="Username"
          value={form.username}
          onChange={(event) => patch({ username: event.target.value })}
        />
        <label className="flex items-center justify-between text-sm text-foreground">
          Active
          <Toggle
            checked={form.is_active}
            onCheckedChange={(checked) => patch({ is_active: checked })}
          />
        </label>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="proxy-pool-save"
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

export { ProxyPoolFormModal };
