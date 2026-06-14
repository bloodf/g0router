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

import { UsersPanel, changePassword } from "./users-panel";

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

  it("renders seeded user rows (no passwords) and a change-password control", () => {
    const { container, root } = mount(<UsersPanel initialUsers={seededUsers} />);
    const html = container.innerHTML;
    expect(html).toContain("admin");
    expect(html).toContain("Administrator");
    expect(container.querySelector('[data-testid="user-row"]')).toBeTruthy();
    expect(container.querySelector('input[aria-label="New password"]')).toBeTruthy();
    expect(html).not.toContain("123456");
    void act(() => root.unmount());
    container.remove();
  });

  it("changePassword PUTs /api/auth/password with both fields", async () => {
    const stub = vi.fn().mockResolvedValue({});
    await changePassword("123456", "newpass789", stub as never);
    const [path, init] = stub.mock.calls[0];
    expect(path).toBe("/api/auth/password");
    expect((init as RequestInit).method).toBe("PUT");
    const body = JSON.parse((init as RequestInit).body as string);
    expect(body.current_password).toBe("123456");
    expect(body.new_password).toBe("newpass789");
  });
});
