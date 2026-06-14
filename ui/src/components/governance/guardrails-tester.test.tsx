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

import { GuardrailsTester } from "./guardrails-tester";

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

  it("submitting POSTs /api/guardrails/test with the typed prompt", async () => {
    apiFetchMock.mockResolvedValue({ blocked: false, redacted_prompt: "hi", matches: [] });
    const { container, root } = mount(<GuardrailsTester />);

    const input = container.querySelector('input[aria-label="Test prompt"]') as HTMLInputElement;
    const setter = Object.getOwnPropertyDescriptor(
      window.HTMLInputElement.prototype,
      "value"
    )!.set!;
    void act(() => {
      setter.call(input, "my secret password");
      input.dispatchEvent(new dom.window.Event("input", { bubbles: true }));
    });

    const testBtn = Array.from(container.querySelectorAll("button")).find((b) =>
      (b.textContent ?? "").includes("Test")
    ) as HTMLButtonElement;
    await act(async () => {
      testBtn.click();
      await Promise.resolve();
    });

    const postCall = apiFetchMock.mock.calls.find(
      ([path, init]) =>
        path === "/api/guardrails/test" && (init as RequestInit)?.method === "POST"
    );
    expect(postCall).toBeTruthy();
    const body = JSON.parse((postCall![1] as RequestInit).body as string);
    expect(body.prompt).toBe("my secret password");

    void act(() => root.unmount());
    container.remove();
  });

  it("renders a blocked result when the response is blocked:true", async () => {
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

  it("renders a not-blocked result when the response is blocked:false", async () => {
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
