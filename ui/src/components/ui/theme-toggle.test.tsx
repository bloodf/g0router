import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { ThemeToggle, nextTheme } from "./theme-toggle";

describe("nextTheme", () => {
  it("cycles light -> dark -> system -> light", () => {
    expect(nextTheme("light")).toBe("dark");
    expect(nextTheme("dark")).toBe("system");
    expect(nextTheme("system")).toBe("light");
  });
});

describe("ThemeToggle", () => {
  it("renders a button with an icon for the current store theme", () => {
    const html = renderToString(<ThemeToggle />);
    expect(html).toContain("<button");
    expect(html).toContain("<svg");
  });

  it("exposes an aria-label that names the current theme", () => {
    const html = renderToString(<ThemeToggle />);
    const labelMatch = html.match(/aria-label="([^"]+)"/);
    expect(labelMatch).not.toBeNull();
    expect(labelMatch![1].toLowerCase()).toMatch(/light|dark|system/);
  });
});
