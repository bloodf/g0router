import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { Team } from "@/lib/types";

export interface TeamFormModalProps {
  open: boolean;
  team: Team | null;
  onClose: () => void;
  onSaved?: () => void;
}

// TeamFormModal creates/edits a team via POST /api/teams (new) or
// PUT /api/teams/{id} (edit). Variant-HAVE against the mock; no Go /api/teams
// exists yet (§8 ESCALATION-1a).
function TeamFormModal({ open, team, onClose, onSaved }: TeamFormModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [budgetUsd, setBudgetUsd] = React.useState("0");
  const [budgetPeriod, setBudgetPeriod] = React.useState("monthly");
  const [rateLimitRpm, setRateLimitRpm] = React.useState("0");
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (team) {
      setName(team.name);
      setBudgetUsd(String(team.budget_usd));
      setBudgetPeriod(team.budget_period);
      setRateLimitRpm(String(team.rate_limit_rpm));
    } else {
      setName("");
      setBudgetUsd("0");
      setBudgetPeriod("monthly");
      setRateLimitRpm("0");
    }
  }, [team]);

  async function save() {
    setBusy(true);
    const payload = {
      name,
      budget_usd: Number(budgetUsd) || 0,
      budget_period: budgetPeriod,
      rate_limit_rpm: Number(rateLimitRpm) || 0,
    };
    try {
      if (team) {
        await apiFetch(`/api/teams/${team.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/teams", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: team ? "Team updated" : "Team created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the team" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={team ? "Edit team" : "New team"}>
      <div className="flex flex-col gap-4">
        <Input
          id="team-name"
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <Input
          id="team-budget-usd"
          label="Budget (USD)"
          type="number"
          value={budgetUsd}
          onChange={(event) => setBudgetUsd(event.target.value)}
        />
        <Select
          id="team-budget-period"
          label="Budget period"
          value={budgetPeriod}
          onChange={(event) => setBudgetPeriod(event.target.value)}
          options={[
            { value: "daily", label: "Daily" },
            { value: "weekly", label: "Weekly" },
            { value: "monthly", label: "Monthly" },
          ]}
        />
        <Input
          id="team-rate-limit-rpm"
          label="Rate limit (RPM)"
          type="number"
          value={rateLimitRpm}
          onChange={(event) => setRateLimitRpm(event.target.value)}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="team-save"
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

export { TeamFormModal };
