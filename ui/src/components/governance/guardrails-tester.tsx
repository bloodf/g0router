import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";

interface GuardrailsTestResult {
  blocked: boolean;
  redacted_prompt: string;
  matches: string[];
}

type ApiFetch = typeof apiFetch;

// runGuardrailsTest is the pure POST seam (chat-window streamChatCompletion
// precedent) so the §1.3 interaction can be unit-tested with a stubbed apiFetch
// without simulating typed input in JSDOM.
export async function runGuardrailsTest(
  prompt: string,
  fetchImpl: ApiFetch = apiFetch
): Promise<GuardrailsTestResult> {
  return fetchImpl<GuardrailsTestResult>("/api/guardrails/test", {
    method: "POST",
    body: JSON.stringify({ prompt }),
  });
}

// GuardrailsTester is the cluster's authoritative interaction surface (plan
// §1.3). It POSTs the typed prompt to /api/guardrails/test and renders the
// {blocked, redacted_prompt, matches} result with literal "Blocked"/"Allowed"
// text. Variant-HAVE against the mock; no Go /api/guardrails/test exists yet
// (§8 ESCALATION-1d).
function GuardrailsTester() {
  const [prompt, setPrompt] = React.useState("");
  const [busy, setBusy] = React.useState(false);
  const [result, setResult] = React.useState<GuardrailsTestResult | null>(null);
  const [failed, setFailed] = React.useState(false);

  async function runTest() {
    setBusy(true);
    setFailed(false);
    try {
      const res = await runGuardrailsTest(prompt);
      setResult(res);
    } catch {
      setFailed(true);
      setResult(null);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-end gap-2">
        <div className="flex-1">
          <Input
            aria-label="Test prompt"
            placeholder="Enter a prompt to test against the guardrails"
            value={prompt}
            onChange={(event) => setPrompt(event.target.value)}
          />
        </div>
        <Button
          data-testid="guardrails-test"
          variant="primary"
          loading={busy}
          onClick={runTest}
        >
          Test
        </Button>
      </div>

      {result ? (
        <div
          data-testid="guardrails-test-result"
          className="rounded-lg border border-border px-4 py-3"
        >
          {result.blocked ? (
            <div className="flex flex-col gap-1">
              <Badge variant="error" size="sm">
                Blocked
              </Badge>
              <p className="text-xs text-muted-foreground">
                Prompt blocked by guardrails. Matched:{" "}
                {result.matches.join(", ") || "—"}
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-1">
              <Badge variant="success" size="sm">
                Allowed
              </Badge>
              <p className="text-xs text-muted-foreground">
                Prompt allowed (not blocked).
              </p>
            </div>
          )}
        </div>
      ) : null}

      {failed ? (
        <p className="text-xs text-destructive">Failed to run the guardrails test.</p>
      ) : null}
    </div>
  );
}

export { GuardrailsTester };
