import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useNotificationStore } from "@/stores/notification";

export interface ProviderNode {
  id: string;
  name: string;
  base_url: string;
  type: string;
  enabled: boolean;
}

export interface ProviderNodeModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

// ProviderNodeModal (PAR-UI-109/110/111) creates an OpenAI-compatible custom
// provider node and validates its endpoint. It consumes the NEW Go provider-nodes
// API (internal/admin/nodes.go). The api_key is sent to /validate transiently and
// is never persisted server-side.
export function ProviderNodeModal({ open, onClose, onCreated }: ProviderNodeModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [nodes, setNodes] = React.useState<ProviderNode[]>([]);
  const [name, setName] = React.useState("");
  const [prefix, setPrefix] = React.useState("");
  const [baseUrl, setBaseUrl] = React.useState("");
  const [apiKey, setApiKey] = React.useState("");
  const [validating, setValidating] = React.useState(false);
  const [valid, setValid] = React.useState<boolean | null>(null);
  const [saving, setSaving] = React.useState(false);

  const loadNodes = React.useCallback(() => {
    apiFetch<{ nodes: ProviderNode[] }>("/api/provider-nodes")
      .then((data) => setNodes(data?.nodes ?? []))
      .catch(() => setNodes([]));
  }, []);

  React.useEffect(() => {
    if (!open) return;
    loadNodes();
  }, [open, loadNodes]);

  async function validate() {
    if (!baseUrl.trim()) return;
    setValidating(true);
    setValid(null);
    try {
      const result = await apiFetch<{ valid: boolean; error?: string }>(
        "/api/provider-nodes/validate",
        {
          method: "POST",
          body: JSON.stringify({
            baseUrl: baseUrl.trim(),
            apiKey: apiKey || undefined,
            type: "openai-compatible",
          }),
        }
      );
      setValid(result?.valid ?? false);
      if (!result?.valid) {
        pushToast({ message: result?.error || "Endpoint validation failed" });
      }
    } catch {
      setValid(false);
      pushToast({ message: "Endpoint validation failed" });
    } finally {
      setValidating(false);
    }
  }

  async function createNode() {
    if (!name.trim() || !baseUrl.trim()) return;
    setSaving(true);
    try {
      await apiFetch<{ node: ProviderNode }>("/api/provider-nodes", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          prefix: prefix.trim() || undefined,
          apiType: "openai",
          baseUrl: baseUrl.trim(),
          type: "openai-compatible",
        }),
      });
      pushToast({ message: "Provider node created" });
      setName("");
      setPrefix("");
      setBaseUrl("");
      setApiKey("");
      setValid(null);
      loadNodes();
      onCreated?.();
    } catch {
      pushToast({ message: "Failed to create the provider node" });
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Custom provider node" size="lg">
      <div className="flex flex-col gap-4">
        {nodes.length > 0 ? (
          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-semibold text-foreground">Existing nodes</h3>
            {nodes.map((node) => (
              <div
                key={node.id}
                data-testid="provider-node-row"
                className="flex items-center justify-between rounded-md border border-border px-3 py-2"
              >
                <div className="flex flex-col">
                  <span className="text-sm font-medium text-foreground">{node.name}</span>
                  <span className="font-mono text-xs text-muted-foreground">{node.base_url}</span>
                </div>
                <Badge variant={node.enabled ? "success" : "neutral"} size="sm">
                  {node.type}
                </Badge>
              </div>
            ))}
          </div>
        ) : null}

        <div className="flex flex-col gap-3">
          <Input
            data-testid="node-name"
            label="Name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="My local model"
          />
          <Input
            label="Prefix"
            value={prefix}
            onChange={(event) => setPrefix(event.target.value)}
            placeholder="local"
          />
          <Input
            data-testid="node-base-url"
            label="Base URL"
            value={baseUrl}
            onChange={(event) => {
              setBaseUrl(event.target.value);
              setValid(null);
            }}
            placeholder="http://localhost:1234/v1"
          />
          <Input
            label="API key (optional, not stored)"
            type="password"
            value={apiKey}
            onChange={(event) => setApiKey(event.target.value)}
          />
        </div>

        {valid !== null ? (
          <Badge variant={valid ? "success" : "error"} size="sm">
            {valid ? "endpoint reachable" : "endpoint invalid"}
          </Badge>
        ) : null}

        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="node-validate"
            variant="outline"
            loading={validating}
            onClick={validate}
          >
            Validate
          </Button>
          <Button
            data-testid="node-create"
            variant="primary"
            loading={saving}
            onClick={createNode}
          >
            Create node
          </Button>
        </div>
      </div>
    </Modal>
  );
}
