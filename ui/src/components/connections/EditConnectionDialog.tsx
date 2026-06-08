import { useMemo, useState, useEffect } from "react";
import { apiFetch } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { toast } from "sonner";
import type { Connection, Provider } from "@/lib/types";

interface EditConnectionDialogProps {
  connection: Connection | null;
  provider: Provider | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSuccess: () => void;
}

export function EditConnectionDialog({
  connection,
  provider,
  open,
  onOpenChange,
  onSuccess,
}: EditConnectionDialogProps) {
  const [name, setName] = useState("");
  const [authType, setAuthType] = useState<Connection["auth_type"]>("api_key");
  const [credential, setCredential] = useState("");
  const [isActive, setIsActive] = useState(true);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (connection) {
      setName(connection.name);
      setAuthType(connection.auth_type);
      setIsActive(connection.is_active);
      setCredential("");
    }
  }, [connection]);

  const credentialLabel = useMemo(() => {
    if (authType === "api_key") return "API key";
    if (authType === "oauth") return "Access token";
    return undefined;
  }, [authType]);

  const needsCredential = authType === "api_key" || authType === "oauth";

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!connection) return;
    if (!name.trim()) {
      toast.error("Connection name is required");
      return;
    }
    if (needsCredential && !credential.trim()) {
      toast.error(`${credentialLabel} is required`);
      return;
    }

    const body: Record<string, unknown> = {
      provider: connection.provider,
      name: name.trim(),
      auth_type: authType,
      is_active: isActive,
    };
    if (authType === "api_key" && credential.trim()) {
      body.api_key = credential.trim();
    }
    if (authType === "oauth" && credential.trim()) {
      body.access_token = credential.trim();
    }

    setBusy(true);
    try {
      await apiFetch(`/api/connections/${connection.id}`, {
        method: "PUT",
        body,
      });
      toast.success("Connection updated");
      onOpenChange(false);
      onSuccess();
    } catch (err: any) {
      toast.error(err?.message || "Failed to update connection");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ProviderIcon
              provider={provider?.id ?? connection?.provider}
              iconUrl={provider?.icon_url}
              size={24}
            />
            Edit connection
          </DialogTitle>
          <DialogDescription>Update connection settings.</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="edit-conn-name">Name</Label>
            <Input
              id="edit-conn-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Primary"
              autoFocus
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="edit-conn-auth">Auth type</Label>
            <Select
              value={authType}
              onValueChange={(v) => setAuthType(v as Connection["auth_type"])}
            >
              <SelectTrigger id="edit-conn-auth" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(provider?.auth_types ?? ["api_key", "oauth", "noauth"]).map((a) => (
                  <SelectItem key={a} value={a}>
                    {a.replace("_", " ")}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {needsCredential && (
            <div className="space-y-1.5">
              <Label htmlFor="edit-conn-credential">{credentialLabel}</Label>
              <Input
                id="edit-conn-credential"
                type="password"
                value={credential}
                onChange={(e) => setCredential(e.target.value)}
                placeholder={credentialLabel}
              />
            </div>
          )}

          <div className="flex items-center justify-between">
            <Label htmlFor="edit-conn-active" className="cursor-pointer">
              Active
            </Label>
            <Switch
              id="edit-conn-active"
              checked={isActive}
              onCheckedChange={setIsActive}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={busy}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? "Saving..." : "Save changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
