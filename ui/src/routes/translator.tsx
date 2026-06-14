import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { TranslatorStep } from "@/components/translator/translator-step";
import { apiFetch } from "@/lib/api";
import { prettyJson } from "@/lib/translator-format";

export const Route = createFileRoute("/translator")({
  component: TranslatorPage,
});

// The 7-step request/response transformation inspector (w6-i §1.6 textarea
// variant). Each step is a plain <textarea> (NO Monaco/CodeMirror). The first
// step loads a sample client request from the mock; Translate transforms it via
// the mock and writes the result into the downstream (OpenAI Intermediate) panel.
const STEP_LABELS = [
  "Client Request",
  "Source Body",
  "OpenAI Intermediate",
  "Target Request",
  "Provider Response",
  "OpenAI Response",
  "Client Response",
] as const;

const STEP_DESCRIPTIONS: Record<string, string> = {
  "Client Request": "Raw request received from the client.",
  "Source Body": "Parsed source request body.",
  "OpenAI Intermediate": "Normalized OpenAI-format intermediate.",
  "Target Request": "Request shaped for the target provider.",
  "Provider Response": "Raw response from the provider.",
  "OpenAI Response": "Normalized OpenAI-format response.",
  "Client Response": "Response returned to the client.",
};

interface LoadResponse {
  file: string;
  payload: string;
}

interface TranslateResponse {
  payload: string;
}

function TranslatorPage() {
  const [steps, setSteps] = React.useState<string[]>(() =>
    STEP_LABELS.map(() => "")
  );
  const [busy, setBusy] = React.useState(false);

  function setStep(index: number, value: string) {
    setSteps((prev) => {
      const next = [...prev];
      next[index] = value;
      return next;
    });
  }

  async function loadSample() {
    setBusy(true);
    try {
      const res = await apiFetch<LoadResponse>(
        "/api/translator/load?file=sample"
      );
      setStep(0, prettyJson(res.payload));
    } finally {
      setBusy(false);
    }
  }

  async function translate() {
    setBusy(true);
    try {
      const res = await apiFetch<TranslateResponse>(
        "/api/translator/translate",
        {
          method: "POST",
          body: JSON.stringify({
            from: "client",
            to: "openai",
            payload: steps[0],
          }),
        }
      );
      // Write the transformed payload into the downstream panel.
      setStep(2, prettyJson(res.payload));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Translator</h1>
        <Button
          type="button"
          data-testid="translator-translate"
          disabled={busy}
          onClick={() => void translate()}
        >
          Translate
        </Button>
      </div>
      <p className="text-sm text-muted-foreground">
        Inspect each step of the request/response transformation pipeline.
      </p>
      <Card padding="sm">
        <div className="grid gap-3">
          {STEP_LABELS.map((label, i) => (
            <TranslatorStep
              key={label}
              label={label}
              description={STEP_DESCRIPTIONS[label]}
              value={steps[i]}
              onChange={(v) => setStep(i, v)}
              onLoad={i === 0 ? () => void loadSample() : undefined}
            />
          ))}
        </div>
      </Card>
    </div>
  );
}
