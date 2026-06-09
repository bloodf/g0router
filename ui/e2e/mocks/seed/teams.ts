import type { Team } from "../../src/lib/types";

export function seedTeams(): Team[] {
  return [
    { id: "team-1", name: "Engineering", budget_usd: 2000, budget_used_usd: 850, budget_period: "monthly", rate_limit_rpm: 5000 },
    { id: "team-2", name: "Data Science", budget_usd: 1500, budget_used_usd: 420, budget_period: "monthly", rate_limit_rpm: 2000 },
  ];
}
