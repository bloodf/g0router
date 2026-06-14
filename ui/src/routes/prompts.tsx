import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Toggle } from "@/components/ui/toggle";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { PromptFormModal } from "@/components/governance/prompt-form-modal";
import { useNotificationStore } from "@/stores/notification";
import type { PromptTemplate } from "@/lib/types";

export const Route = createFileRoute("/prompts")({
  component: PromptsPage,
});

// PromptsPage (PAR-UI-130 subset) lists prompt templates from
// GET /api/prompt-templates and drives create/edit (PromptFormModal), delete
// (ConfirmModal), and the is_active Toggle. Variant-HAVE against the mock; no Go
// /api/prompt-templates exists yet (§8 ESCALATION-1e).
function PromptsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [prompts, setPrompts] = React.useState<PromptTemplate[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<PromptTemplate | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<PromptTemplate | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<PromptTemplate[]>("/api/prompt-templates")
      .then((rows) => {
        setPrompts(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setPrompts([]);
        setLoading(false);
        pushToast({ message: "Failed to load prompts" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setActive(prompt: PromptTemplate, active: boolean) {
    setPrompts((prev) =>
      prev.map((p) => (p.id === prompt.id ? { ...p, is_active: active } : p))
    );
    try {
      await apiFetch(`/api/prompt-templates/${prompt.id}`, {
        method: "PUT",
        body: JSON.stringify({ ...prompt, is_active: active }),
      });
    } catch {
      pushToast({ message: "Failed to update the prompt" });
      load();
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/prompt-templates/${deleting.id}`, { method: "DELETE" });
      setPrompts((prev) => prev.filter((p) => p.id !== deleting.id));
      pushToast({ message: "Prompt deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the prompt" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Prompts</h1>
        <Button
          data-testid="prompt-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New prompt
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : prompts.length === 0 ? (
        <p className="text-sm text-muted-foreground">No prompt templates yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {prompts.map((prompt) => (
            <div
              key={prompt.id}
              data-testid="prompt-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div className="flex flex-col gap-1">
                <p className="text-sm font-medium text-foreground">{prompt.name}</p>
                <div className="flex flex-wrap items-center gap-1">
                  {prompt.models.map((model) => (
                    <Badge key={model} variant="neutral" size="sm">
                      {model}
                    </Badge>
                  ))}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Toggle
                  checked={prompt.is_active}
                  onCheckedChange={(checked) => setActive(prompt, checked)}
                  aria-label={`Toggle ${prompt.name}`}
                />
                <Button variant="ghost" size="sm" onClick={() => setEditing(prompt)}>
                  Edit
                </Button>
                <Button
                  data-testid="prompt-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(prompt)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <PromptFormModal
        open={creating || editing !== null}
        prompt={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete prompt"
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
