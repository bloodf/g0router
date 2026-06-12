import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { JSDOM } from "jsdom";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import React from "react";
import { I18nProvider, useI18n } from "./i18n";
import i18n from "../i18n";

const dom = new JSDOM("<!DOCTYPE html><html><body></body></html>", {
  url: "http://localhost:20129",
});

global.window = dom.window as unknown as Window & typeof globalThis;
global.document = dom.window.document;
Object.defineProperty(globalThis, "navigator", {
  value: dom.window.navigator,
  configurable: true,
});

const { subscribe } = vi.hoisted(() => ({
  subscribe: vi.fn(() => vi.fn()),
}));

vi.mock("@tanstack/react-router", () => ({
  useRouter: () => ({ subscribe }),
}));

function render(ui: React.ReactElement) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  void act(() => root.render(ui));
  return { container, root, cleanup };

  function cleanup() {
    void act(() => root.unmount());
    container.remove();
  }
}

describe("I18nProvider", () => {
  beforeEach(async () => {
    subscribe.mockClear();
    document.cookie = "";
    await i18n.changeLanguage("en");
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders children", () => {
    const { container, cleanup } = render(
      <I18nProvider>
        <div data-testid="child">hello</div>
      </I18nProvider>
    );
    expect(container.textContent).toContain("hello");
    cleanup();
  });

  it("exposes currentLocale through useI18n", () => {
    let captured: ReturnType<typeof useI18n> | undefined;
    const Capture = () => {
      captured = useI18n();
      return <span>{captured.currentLocale}</span>;
    };
    const { container, cleanup } = render(
      <I18nProvider>
        <Capture />
      </I18nProvider>
    );
    expect(captured?.currentLocale).toBe("en");
    expect(container.textContent).toContain("en");
    cleanup();
  });

  it("subscribes to router route changes", () => {
    render(
      <I18nProvider>
        <div />
      </I18nProvider>
    );
    expect(subscribe).toHaveBeenCalledWith("onResolved", expect.any(Function));
  });

  it("setLocale calls apiFetch and updates i18n.language", async () => {
    let setLocaleFn: ((code: string) => Promise<void>) | undefined;
    const Capture = () => {
      const ctx = useI18n();
      setLocaleFn = ctx.setLocale;
      return null;
    };

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ data: null, error: null }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const { cleanup } = render(
      <I18nProvider>
        <Capture />
      </I18nProvider>
    );

    expect(setLocaleFn).toBeDefined();
    expect(i18n.language).toBe("en");

    await act(async () => {
      await setLocaleFn!("de");
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://localhost:20129/api/locale");
    expect((init as RequestInit).method).toBe("POST");
    expect(JSON.parse((init as RequestInit).body as string)).toEqual({
      locale: "de",
    });
    expect(i18n.language).toBe("de");

    cleanup();
  });
});
