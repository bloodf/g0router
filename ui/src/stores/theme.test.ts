import { describe, it, expect, beforeEach, vi } from "vitest";

function setupMocks(prefersDark = false, storage: Record<string, string> = {}) {
  const classes = new Set<string>();
  globalThis.document = {
    documentElement: {
      classList: {
        add: (c: string) => classes.add(c),
        remove: (c: string) => classes.delete(c),
        contains: (c: string) => classes.has(c),
        toggle: (c: string, force?: boolean) => {
          if (force === true) {
            classes.add(c);
            return true;
          }
          if (force === false) {
            classes.delete(c);
            return false;
          }
          const has = classes.has(c);
          if (has) classes.delete(c);
          else classes.add(c);
          return !has;
        },
      },
    },
  } as unknown as Document;

  globalThis.localStorage = {
    getItem: (key: string) => storage[key] ?? null,
    setItem: (key: string, value: string) => {
      storage[key] = value;
    },
    removeItem: (key: string) => {
      delete storage[key];
    },
  } as Storage;

  globalThis.matchMedia = vi.fn((query: string) => ({
    matches: prefersDark && query === "(prefers-color-scheme: dark)",
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia;
}

describe("initTheme", () => {
  beforeEach(() => {
    vi.resetModules();
    setupMocks();
  });

  it("sets dark class when theme=dark", async () => {
    const storage = {
      theme: JSON.stringify({ state: { theme: "dark" }, version: 0 }),
    };
    setupMocks(false, storage);
    const { initTheme } = await import("./theme");
    initTheme();
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("removes dark class when theme=light", async () => {
    const storage = {
      theme: JSON.stringify({ state: { theme: "light" }, version: 0 }),
    };
    setupMocks(true, storage);
    const { initTheme } = await import("./theme");
    initTheme();
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("follows system when theme=system and prefers dark", async () => {
    const storage = {
      theme: JSON.stringify({ state: { theme: "system" }, version: 0 }),
    };
    setupMocks(true, storage);
    const { initTheme } = await import("./theme");
    initTheme();
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });
});
