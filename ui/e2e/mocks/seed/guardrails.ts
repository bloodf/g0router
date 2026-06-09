import type { Guardrails } from "../../src/lib/types";

export function seedGuardrails(): Guardrails {
  return {
    guardrails_enabled: false,
    guardrails_blocklist: ["badword1", "badword2"],
    pii_redaction_enabled: false,
    pii_redaction_types: ["email", "phone", "ssn"],
  };
}
