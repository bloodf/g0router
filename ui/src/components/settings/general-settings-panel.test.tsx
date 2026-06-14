import { describe, it, expect, vi, beforeEach } from "vitest";
import { JSDOM } from "jsdom";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { renderToString } from "react-dom/server";
import React from "react";

const dom = new JSDOM("<!DOCTYPE html><html><body></body></html>", {
  url: "http://localhost:20133",
});
global.window = dom.window as unknown as Window & typeof globalThis;
global.document = dom.window.document;
Object.defineProperty(globalThis, "navigator", {
  value: dom.window.navigator,
  configurable: true,
});
// Radix Switch needs ResizeObserver/matchMedia; stub minimally.
if (!global.window.matchMedia) {
  // @ts-expect-error test stub
  global.window.matchMedia = () => ({ matches: false, addEventListener() {}, removeEventListener() {} });
}

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));
vi.mock("@/lib/api", () => ({
  apiFetch: apiFetchMock,
  ApiError: class ApiError extends Error {},
}));

import { GeneralSettingsPanel } from "./general-settings-panel";

describe("GeneralSettingsPanel", () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
  });

  it("renders the require_login toggle and a Save button reflecting seeded settings", () => {
    const html = renderToString(
      <GeneralSettingsPanel initialSettings={{ require_login: true, theme: "system" }} />
    );
    expect(html).toContain("Require login");
    expect(html).toContain('role="switch"');
    expect(html).toContain("Save");
    expect(html).toContain("theme-segmented");
  });

  it("Save PUTs /api/settings with the toggled require_login key", async () => {
    apiFetchMock.mockResolvedValue({ require_login: false });
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    void act(() =>
      root.render(
        <GeneralSettingsPanel initialSettings={{ require_login: true, theme: "system" }} />
      )
    );

    // Toggle require_login off, then click Save.
    const toggle = container.querySelector('button[role="switch"]') as HTMLButtonElement;
    expect(toggle).toBeTruthy();
    void act(() => toggle.click());

    const saveBtn = Array.from(container.querySelectorAll("button")).find((b) =>
      (b.textContent ?? "").includes("Save")
    ) as HTMLButtonElement;
    expect(saveBtn).toBeTruthy();
    await act(async () => {
      saveBtn.click();
      await Promise.resolve();
    });

    expect(apiFetchMock).toHaveBeenCalled();
    const putCall = apiFetchMock.mock.calls.find(
      ([path, init]) => path === "/api/settings" && (init as RequestInit)?.method === "PUT"
    );
    expect(putCall).toBeTruthy();
    const body = JSON.parse((putCall![1] as RequestInit).body as string);
    expect(body.require_login).toBe(false);

    void act(() => root.unmount());
    container.remove();
  });
});
