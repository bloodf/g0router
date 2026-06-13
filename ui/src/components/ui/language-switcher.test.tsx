import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { LanguageSwitcher, DEFAULT_LOCALES } from "./language-switcher";

describe("LanguageSwitcher", () => {
  it("renders a trigger showing the current flag", () => {
    const html = renderToString(<LanguageSwitcher />);
    expect(html).toContain("<button");
    expect(html).toContain(DEFAULT_LOCALES[0].flag);
  });

  it("renders one grid button per DEFAULT_LOCALES entry when open", () => {
    const html = renderToString(<LanguageSwitcher defaultOpen />);
    const gridButtons = (html.match(/data-testid="locale-option"/g) ?? []).length;
    expect(gridButtons).toBe(DEFAULT_LOCALES.length);
  });

  it("respects a custom locales prop", () => {
    const custom = [
      { code: "en", flag: "🇬🇧", label: "English" },
      { code: "fr", flag: "🇫🇷", label: "Français" },
    ];
    const html = renderToString(
      <LanguageSwitcher defaultOpen locales={custom} />
    );
    const gridButtons = (html.match(/data-testid="locale-option"/g) ?? []).length;
    expect(gridButtons).toBe(2);
  });

  it("exposes aria-haspopup=dialog and an accessible label on the trigger", () => {
    const html = renderToString(<LanguageSwitcher />);
    expect(html).toContain('aria-haspopup="dialog"');
    expect(html).toMatch(/aria-label="[^"]+"/);
  });
});
