import { describe, it, expect, vi, beforeEach } from "vitest";
import { JSDOM } from "jsdom";
import { act } from "react";
import { createRoot } from "react-dom/client";
import React from "react";

const dom = new JSDOM("<!DOCTYPE html><html><body></body></html>", {
  url: "http://localhost:20145",
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

import { UsersPanel } from "./users-panel";

const seededUsers = [
  { id: "user-1", username: "admin", display_name: "Administrator", role: "admin" },
];

function mount(node: React.ReactElement) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  void act(() => root.render(node));
  return { container, root };
}

describe("UsersPanel", () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
  });

  it("renders seeded user rows (no passwords)", () => {
    const { container, root } = mount(<UsersPanel initialUsers={seededUsers} />);
    const html = container.innerHTML;
    expect(html).toContain("admin");
    expect(html).toContain("Administrator");
    expect(container.querySelector('[data-testid="user-row"]')).toBeTruthy();
    void act(() => root.unmount());
    container.remove();
  });

  it("change-password submit PUTs /api/auth/password with both fields", async () => {
    apiFetchMock.mockResolvedValue({});
    const { container, root } = mount(<UsersPanel initialUsers={seededUsers} />);

    const setter = Object.getOwnPropertyDescriptor(
      window.HTMLInputElement.prototype,
      "value"
    )!.set!;
    const current = container.querySelector(
      'input[aria-label="Current password"]'
    ) as HTMLInputElement;
    const next = container.querySelector(
      'input[aria-label="New password"]'
    ) as HTMLInputElement;
    expect(current).toBeTruthy();
    expect(next).toBeTruthy();
    void act(() => {
      setter.call(current, "123456");
      current.dispatchEvent(new dom.window.Event("input", { bubbles: true }));
      setter.call(next, "newpass789");
      next.dispatchEvent(new dom.window.Event("input", { bubbles: true }));
    });

    const saveBtn = Array.from(container.querySelectorAll("button")).find((b) =>
      (b.textContent ?? "").includes("Change password")
    ) as HTMLButtonElement;
    expect(saveBtn).toBeTruthy();
    await act(async () => {
      saveBtn.click();
      await Promise.resolve();
    });

    const putCall = apiFetchMock.mock.calls.find(
      ([path, init]) =>
        path === "/api/auth/password" && (init as RequestInit)?.method === "PUT"
    );
    expect(putCall).toBeTruthy();
    const body = JSON.parse((putCall![1] as RequestInit).body as string);
    expect(body.current_password).toBe("123456");
    expect(body.new_password).toBe("newpass789");

    void act(() => root.unmount());
    container.remove();
  });
});
