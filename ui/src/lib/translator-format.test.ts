import { describe, it, expect } from "vitest";
import { detectFormat, prettyJson } from "./translator-format";

describe("detectFormat", () => {
  it("extracts provider and model from a JSON payload", () => {
    const out = detectFormat(
      JSON.stringify({ provider: "openai", model: "gpt-4o" })
    );
    expect(out.provider).toBe("openai");
    expect(out.model).toBe("gpt-4o");
    expect(out.sourceFormat).toBe("json");
  });

  it("reports text format for non-JSON input without throwing", () => {
    const out = detectFormat("hello world, not json");
    expect(out.sourceFormat).toBe("text");
    expect(out.provider).toBeUndefined();
    expect(out.model).toBeUndefined();
  });

  it("handles empty input gracefully", () => {
    const out = detectFormat("");
    expect(out.sourceFormat).toBe("text");
  });

  it("is pure — repeated calls give equal results", () => {
    const payload = JSON.stringify({ provider: "anthropic", model: "claude" });
    expect(detectFormat(payload)).toEqual(detectFormat(payload));
  });
});

describe("prettyJson", () => {
  it("formats valid JSON with 2-space indentation", () => {
    const out = prettyJson('{"a":1,"b":2}');
    expect(out).toBe('{\n  "a": 1,\n  "b": 2\n}');
  });

  it("returns the original string unchanged for invalid JSON", () => {
    const input = "not { valid json";
    expect(prettyJson(input)).toBe(input);
  });

  it("is idempotent on already-pretty JSON", () => {
    const once = prettyJson('{"a":1}');
    expect(prettyJson(once)).toBe(once);
  });
});
