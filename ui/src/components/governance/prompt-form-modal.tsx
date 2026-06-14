import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toggle } from "@/components/ui/toggle";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { PromptTemplate } from "@/lib/types";

export interface PromptFormModalProps {
  open: boolean;
  prompt: PromptTemplate | null;
  onClose: () => void;
  onSaved?: () => void;
}

// PromptFormModal creates/edits a prompt template via POST /api/prompt-templates
// (new) or PUT /api/prompt-templates/{id} (edit). Variant-HAVE against the mock;
// no Go /api/prompt-templates exists yet (§8 ESCALATION-1e).
function PromptFormModal({ open, prompt, onClose, onSaved }: PromptFormModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [systemPrompt, setSystemPrompt] = React.useState("");
  const [modelsText, setModelsText] = React.useState("");
  const [isActive, setIsActive] = React.useState(true);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (prompt) {
      setName(prompt.name);
      setSystemPrompt(prompt.system_prompt);
      setModelsText(prompt.models.join(", "));
      setIsActive(prompt.is_active);
    } else {
      setName("");
      setSystemPrompt("");
      setModelsText("");
      setIsActive(true);
    }
  }, [prompt]);

  async function save() {
    setBusy(true);
    const payload = {
      name,
      system_prompt: systemPrompt,
      models: modelsText
        .split(",")
        .map((entry) => entry.trim())
        .filter(Boolean),
      is_active: isActive,
    };
    try {
      if (prompt) {
        await apiFetch(`/api/prompt-templates/${prompt.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/prompt-templates", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: prompt ? "Prompt updated" : "Prompt created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the prompt" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={prompt ? "Edit prompt" : "New prompt"}>
      <div className="flex flex-col gap-4">
        <Input
          id="prompt-name"
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <div className="flex flex-col gap-1.5">
          <label
            htmlFor="prompt-system"
            className="text-sm font-medium text-foreground"
          >
            System prompt
          </label>
          <textarea
            id="prompt-system"
            rows={4}
            value={systemPrompt}
            onChange={(event) => setSystemPrompt(event.target.value)}
            className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          />
        </div>
        <Input
          id="prompt-models"
          label="Models (comma-separated)"
          value={modelsText}
          onChange={(event) => setModelsText(event.target.value)}
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
            data-testid="prompt-save"
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

export { PromptFormModal };
