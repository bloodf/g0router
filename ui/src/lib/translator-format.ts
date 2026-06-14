// Pure, deterministic translator helpers (w6-i §1.6 point 4). No DOM. The
// authoritative translator-logic proof; the textarea wiring is covered by the
// translator e2e.

export interface DetectedFormat {
  provider?: string;
  model?: string;
  sourceFormat: "json" | "text";
  targetFormat?: string;
}

// Inspects a payload string and reports the detected format plus any provider/
// model it can read from a JSON body. Non-JSON input is reported as "text"
// without throwing.
export function detectFormat(payload: string): DetectedFormat {
  const trimmed = payload.trim();
  if (!trimmed) return { sourceFormat: "text" };
  try {
    const parsed = JSON.parse(trimmed) as Record<string, unknown>;
    const provider =
      typeof parsed.provider === "string" ? parsed.provider : undefined;
    const model = typeof parsed.model === "string" ? parsed.model : undefined;
    return { sourceFormat: "json", provider, model, targetFormat: "openai" };
  } catch {
    return { sourceFormat: "text" };
  }
}

// Pretty-prints valid JSON with 2-space indentation; returns the original string
// unchanged when the input is not valid JSON.
export function prettyJson(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}
