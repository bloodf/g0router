import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { ProviderLimits } from "./provider-limits";
import type { Quota } from "@/lib/types";

const quotas: Quota[] = [
  {
    connection_id: "conn-1",
    provider: "openai",
    connection_name: "OpenAI Prod",
    account_label: "org-123",
    plan: "pro",
    used: 45000,
    limit: 100000,
    unit: "tokens",
    reset_at: new Date(Date.now() + 86400000).toISOString(),
    is_active: true,
  },
  {
    connection_id: "conn-5",
    provider: "groq",
    connection_name: "Groq Fast",
    plan: "free",
    used: 8000,
    limit: 0,
    unit: "tokens",
    reset_at: new Date(Date.now() + 86400000).toISOString(),
    is_active: true,
  },
];

describe("ProviderLimits", () => {
  it("renders used/limit and a plan badge for a bounded quota", () => {
    const html = renderToString(<ProviderLimits quotas={[quotas[0]]} />);
    expect(html).toContain("quota-card");
    expect(html).toContain("quota-progress");
    expect(html).toContain("OpenAI Prod");
    expect(html).toContain("pro");
    // used and limit appear (45,000 / 100,000 tokens).
    expect(html).toMatch(/45[,.]?000/);
    expect(html).toMatch(/100[,.]?000/);
  });

  it("renders 'unlimited' when limit is 0", () => {
    const html = renderToString(<ProviderLimits quotas={[quotas[1]]} />);
    expect(html.toLowerCase()).toContain("unlimited");
  });
});
