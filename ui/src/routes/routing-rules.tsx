import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Toggle } from "@/components/ui/toggle";
import { ProviderIcon } from "@/components/ui/provider-icon";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { RoutingRuleModal } from "@/components/routing/routing-rule-modal";
import { useNotificationStore } from "@/stores/notification";
import type { RoutingRule } from "@/lib/types";

export const Route = createFileRoute("/routing-rules")({
  component: RoutingRulesPage,
});

// RoutingRulesPage (PAR-UI-130 subset) lists routing rules from
// GET /api/routing-rules and drives create/edit (RoutingRuleModal) and delete
// (ConfirmModal). Variant-HAVE against the mock; no Go backend yet (§8
// ESCALATION-3a).
function RoutingRulesPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [rules, setRules] = React.useState<RoutingRule[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<RoutingRule | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<RoutingRule | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<RoutingRule[]>("/api/routing-rules")
      .then((rows) => {
        setRules(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setRules([]);
        setLoading(false);
        pushToast({ message: "Failed to load routing rules" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setActive(rule: RoutingRule, active: boolean) {
    setRules((prev) =>
      prev.map((r) => (r.id === rule.id ? { ...r, is_active: active } : r))
    );
    try {
      await apiFetch(`/api/routing-rules/${rule.id}`, {
        method: "PUT",
        body: JSON.stringify({ ...rule, is_active: active }),
      });
    } catch {
      pushToast({ message: "Failed to update the rule" });
      load();
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/routing-rules/${deleting.id}`, { method: "DELETE" });
      setRules((prev) => prev.filter((r) => r.id !== deleting.id));
      pushToast({ message: "Rule deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the rule" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Routing Rules</h1>
        <Button
          data-testid="routing-rule-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New rule
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : rules.length === 0 ? (
        <p className="text-sm text-muted-foreground">No routing rules yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {rules.map((rule) => (
            <div
              key={rule.id}
              data-testid="routing-rule-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div className="flex items-center gap-3">
                <Badge variant="neutral" size="sm">
                  #{rule.priority}
                </Badge>
                <div>
                  <p className="text-sm font-medium text-foreground">{rule.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {rule.cond_field} {rule.cond_operator} {rule.cond_value}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <ProviderIcon
                    slug={rule.target_provider}
                    name={rule.target_provider}
                    size="sm"
                  />
                  {rule.target_provider}
                </span>
                <Toggle
                  checked={rule.is_active}
                  onCheckedChange={(checked) => setActive(rule, checked)}
                  aria-label={`Toggle ${rule.name}`}
                />
                <Button variant="ghost" size="sm" onClick={() => setEditing(rule)}>
                  Edit
                </Button>
                <Button
                  data-testid="routing-rule-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(rule)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <RoutingRuleModal
        open={creating || editing !== null}
        rule={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete rule"
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
