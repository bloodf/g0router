import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { TeamFormModal } from "@/components/governance/team-form-modal";
import { UsersPanel } from "@/components/governance/users-panel";
import { useNotificationStore } from "@/stores/notification";
import type { Team } from "@/lib/types";

export const Route = createFileRoute("/teams")({
  component: TeamsPage,
});

// TeamsPage (PAR-UI-130 subset) lists teams from GET /api/teams and drives
// create/edit (TeamFormModal) and delete (ConfirmModal). It also embeds the
// Users panel (PAR-UI-132, §1.5). Variant-HAVE against the mock; no Go
// /api/teams exists yet (§8 ESCALATION-1a / ESCALATION-2).
function TeamsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [teams, setTeams] = React.useState<Team[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<Team | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<Team | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<Team[]>("/api/teams")
      .then((rows) => {
        setTeams(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setTeams([]);
        setLoading(false);
        pushToast({ message: "Failed to load teams" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/teams/${deleting.id}`, { method: "DELETE" });
      setTeams((prev) => prev.filter((t) => t.id !== deleting.id));
      pushToast({ message: "Team deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the team" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-6">
        <header className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold text-foreground">Teams</h1>
          <Button
            data-testid="team-new"
            variant="primary"
            size="sm"
            onClick={() => setCreating(true)}
          >
            New team
          </Button>
        </header>

        {loading ? (
          <CardSkeleton />
        ) : teams.length === 0 ? (
          <p className="text-sm text-muted-foreground">No teams yet.</p>
        ) : (
          <div className="flex flex-col gap-2">
            {teams.map((team) => (
              <div
                key={team.id}
                data-testid="team-row"
                className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
              >
                <div>
                  <p className="text-sm font-medium text-foreground">{team.name}</p>
                  <p className="text-xs text-muted-foreground">
                    ${team.budget_used_usd} / ${team.budget_usd} · {team.rate_limit_rpm} RPM
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant="neutral" size="sm">
                    {team.budget_period}
                  </Badge>
                  <Button variant="ghost" size="sm" onClick={() => setEditing(team)}>
                    Edit
                  </Button>
                  <Button
                    data-testid="team-delete"
                    variant="danger"
                    size="sm"
                    onClick={() => setDeleting(team)}
                  >
                    Delete
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <UsersPanel />

      <TeamFormModal
        open={creating || editing !== null}
        team={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete team"
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
