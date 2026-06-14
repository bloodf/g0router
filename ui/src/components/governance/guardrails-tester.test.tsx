import { describe, it, expect, vi, beforeEach } from "vitest";
import { JSDOM } from "jsdom";
import { act } from "react";
import { createRoot } from "react-dom/client";
import React from "react";

const dom = new JSDOM("<!DOCTYPE html><html><body></body></html>", {
  url: "http://localhost:20144",
});
global.window = dom.window as unknown as Window & typeof globalThis;
global.document = dom.window.document;
Object.defineProperty(globalThis, "navigator", {
  value: dom.window.navigator,
  configurable: true,
});
if (!global.window.matchMedia) {
  // @ts-expect-error test stub
  global.window.matchMedia = () => ({ matches: false, addEventListener() {}, removeEventListener() {} });
}

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));
vi.mock("@/lib/api", () => ({
  apiFetch: apiFetchMock,
  ApiError: class ApiError extends Error {},
}));

import { GuardrailsTester, runGuardrailsTest } from "./guardrails-tester";

function mount(node: React.ReactElement) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  void act(() => root.render(node));
  return { container, root };
}

describe("GuardrailsTester", () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
  });

  it("renders the test prompt input and a Test button", () => {
    const { container, root } = mount(<GuardrailsTester />);
    expect(container.querySelector('input[aria-label="Test prompt"]')).toBeTruthy();
    const testBtn = Array.from(container.querySelectorAll("button")).find((b) =>
      (b.textContent ?? "").includes("Test")
    );
    expect(testBtn).toBeTruthy();
    void act(() => root.unmount());
    container.remove();
  });

  it("runGuardrailsTest POSTs /api/guardrails/test with the typed prompt", async () => {
    const stub = vi
      .fn()
      .mockResolvedValue({ blocked: false, redacted_prompt: "x", matches: [] });
    await runGuardrailsTest("my secret password", stub as never);
    const [path, init] = stub.mock.calls[0];
    expect(path).toBe("/api/guardrails/test");
    expect((init as RequestInit).method).toBe("POST");
    const body = JSON.parse((init as RequestInit).body as string);
    expect(body.prompt).toBe("my secret password");
  });

  it("renders a blocked result when the Test response is blocked:true", async () => {
    apiFetchMock.mockResolvedValue({
      blocked: true,
      redacted_prompt: "my secret password",
      matches: ["password", "secret"],
    });
    const { container, root } = mount(<GuardrailsTester />);

    const testBtn = Array.from(container.querySelectorAll("button")).find((b) =>
      (b.textContent ?? "").includes("Test")
    ) as HTMLButtonElement;
    await act(async () => {
      testBtn.click();
      await Promise.resolve();
    });

    expect(container.innerHTML).toMatch(/blocked/i);

    void act(() => root.unmount());
    container.remove();
  });

  it("renders a not-blocked result when the Test response is blocked:false", async () => {
    apiFetchMock.mockResolvedValue({ blocked: false, redacted_prompt: "ok", matches: [] });
    const { container, root } = mount(<GuardrailsTester />);

    const testBtn = Array.from(container.querySelectorAll("button")).find((b) =>
      (b.textContent ?? "").includes("Test")
    ) as HTMLButtonElement;
    await act(async () => {
      testBtn.click();
      await Promise.resolve();
    });

    expect(container.innerHTML).toMatch(/allowed|not blocked|clear|passed/i);

    void act(() => root.unmount());
    container.remove();
  });
});
