import { describe, it, expect, vi, beforeEach } from "vitest";
import { JSDOM } from "jsdom";
import { act } from "react";
import { createRoot } from "react-dom/client";
import React from "react";

const dom = new JSDOM("<!DOCTYPE html><html><body></body></html>", {
  url: "http://localhost:20131",
});
global.window = dom.window as unknown as Window & typeof globalThis;
global.document = dom.window.document;
Object.defineProperty(globalThis, "navigator", {
  value: dom.window.navigator,
  configurable: true,
});

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));
vi.mock("@/lib/api", () => ({
  apiFetch: apiFetchMock,
  ApiError: class ApiError extends Error {},
}));

import { useVersionCheck } from "./use-version-check";
import { useSettingsStore } from "@/stores/settings";

function renderHook<T>(hook: () => T): { result: { current: T | undefined }; cleanup: () => void } {
  const result: { current: T | undefined } = { current: undefined };
  function Probe() {
    result.current = hook();
    return null;
  }
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  void act(() => root.render(React.createElement(Probe)));
  return {
    result,
    cleanup: () => {
      void act(() => root.unmount());
      container.remove();
    },
  };
}

describe("useVersionCheck", () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
    useSettingsStore.setState({ updateAvailable: false, latestVersion: null });
  });

  it("fetches /api/version and returns the version", async () => {
    apiFetchMock.mockResolvedValue({
      version: "1.2.3",
      build_date: "2026-06-14",
      update_available: false,
      latest_version: "",
    });

    let view!: ReturnType<typeof renderHook<ReturnType<typeof useVersionCheck>>>;
    await act(async () => {
      view = renderHook(() => useVersionCheck());
      await Promise.resolve();
    });

    expect(apiFetchMock).toHaveBeenCalledWith("/api/version");
    expect(view.result.current?.version).toBe("1.2.3");
    expect(view.result.current?.buildDate).toBe("2026-06-14");
    view.cleanup();
  });

  it("calls setUpdateInfo when update_available is true", async () => {
    apiFetchMock.mockResolvedValue({
      version: "1.2.3",
      build_date: "2026-06-14",
      update_available: true,
      latest_version: "v9.9.9",
    });

    let view!: ReturnType<typeof renderHook<ReturnType<typeof useVersionCheck>>>;
    await act(async () => {
      view = renderHook(() => useVersionCheck());
      await Promise.resolve();
    });

    const state = useSettingsStore.getState();
    expect(state.updateAvailable).toBe(true);
    expect(state.latestVersion).toBe("v9.9.9");
    expect(view.result.current?.updateAvailable).toBe(true);
    expect(view.result.current?.latestVersion).toBe("v9.9.9");
    view.cleanup();
  });

  it("does not set the badge when update_available is false", async () => {
    apiFetchMock.mockResolvedValue({
      version: "1.2.3",
      build_date: "2026-06-14",
      update_available: false,
      latest_version: "",
    });

    let view!: ReturnType<typeof renderHook<ReturnType<typeof useVersionCheck>>>;
    await act(async () => {
      view = renderHook(() => useVersionCheck());
      await Promise.resolve();
    });

    const state = useSettingsStore.getState();
    expect(state.updateAvailable).toBe(false);
    expect(state.latestVersion).toBe(null);
    view.cleanup();
  });
});
