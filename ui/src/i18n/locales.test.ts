import { describe, it, expect } from "vitest";
import { LOCALES } from "./locales";

const localeCodes = LOCALES.map((l) => l.code);

describe("LOCALES catalog", () => {
  it("has exactly 33 entries", () => {
    expect(LOCALES.length).toBe(33);
  });

  it("has unique codes", () => {
    expect(new Set(localeCodes).size).toBe(localeCodes.length);
  });

  it.each(LOCALES)(
    "$code has non-empty code, name, and flag and a matching JSON file",
    async (locale) => {
      expect(locale.code).toBeTruthy();
      expect(locale.name).toBeTruthy();
      expect(locale.flag).toBeTruthy();
      const resources = await import(`./locales/${locale.code}.json`);
      expect(resources).toBeDefined();
    }
  );
});
