import type { Guardrails } from "../../src/lib/types";

export function seedGuardrails(): Guardrails {
  // w6-k path B (plan §1.3): the guardrails prompt-tester spec
  // (guardrails.spec.ts) types "my secret password" and asserts the result
  // shows /blocked/i. The mock /api/guardrails/test handler returns blocked:true
  // only when guardrails_enabled is true AND a blocklist word is a substring of
  // the prompt — so the seed is corrected to a deterministically-blocking state.
  // This seed is consumed only by the guardrails surface in this cluster.
  return {
    guardrails_enabled: true,
    guardrails_blocklist: ["password", "secret", "badword1"],
    pii_redaction_enabled: false,
    pii_redaction_types: ["email", "phone", "ssn"],
  };
}
