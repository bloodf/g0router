import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toggle } from "@/components/ui/toggle";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { McpTool, McpToolGroup } from "@/lib/types";

export interface McpToolGroupModalProps {
  open: boolean;
  group: McpToolGroup | null;
  tools: McpTool[];
  onClose: () => void;
  onSaved?: () => void;
}

// McpToolGroupModal (PAR-UI-130 /mcp/tools) creates/edits an MCP tool-group via
// POST /api/mcp/tool-groups (new) or PUT /api/mcp/tool-groups/{id} (edit). Uses
// snake_case keys to match the tool-groups mock shape (§1.2/§1.4). Variant-HAVE;
// no Go backend yet (§8 ESC-1b).
function McpToolGroupModal({
  open,
  group,
  tools,
  onClose,
  onSaved,
}: McpToolGroupModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [toolIds, setToolIds] = React.useState<string[]>([]);
  const [isActive, setIsActive] = React.useState(true);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (group) {
      setName(group.name);
      setToolIds(group.tool_ids);
      setIsActive(group.is_active);
    } else {
      setName("");
      setToolIds([]);
      setIsActive(true);
    }
  }, [group]);

  function toggleTool(toolName: string, checked: boolean) {
    setToolIds((prev) =>
      checked ? [...prev, toolName] : prev.filter((id) => id !== toolName)
    );
  }

  async function save() {
    setBusy(true);
    const payload = { name, tool_ids: toolIds, is_active: isActive };
    try {
      if (group) {
        await apiFetch(`/api/mcp/tool-groups/${group.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/mcp/tool-groups", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: group ? "Tool group updated" : "Tool group created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the tool group" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={group ? "Edit tool group" : "New tool group"}
    >
      <div className="flex flex-col gap-4">
        <Input
          id="mcp-tool-group-name"
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-foreground">Tools</span>
          {tools.length === 0 ? (
            <p className="text-xs text-muted-foreground">No tools available.</p>
          ) : (
            tools.map((tool) => (
              <label
                key={tool.function.name}
                className="flex items-center gap-2 text-sm text-foreground"
              >
                <input
                  type="checkbox"
                  checked={toolIds.includes(tool.function.name)}
                  onChange={(event) =>
                    toggleTool(tool.function.name, event.target.checked)
                  }
                />
                {tool.function.name}
              </label>
            ))
          )}
        </div>
        <label className="flex items-center justify-between text-sm text-foreground">
          Active
          <Toggle checked={isActive} onCheckedChange={setIsActive} />
        </label>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="mcp-tool-group-save"
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

export { McpToolGroupModal };
