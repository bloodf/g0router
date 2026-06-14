import { describe, it, expect } from "vitest";
import { groupSkillsByCategory } from "./skills-format";
import type { Skill } from "./types";

const sample: Skill[] = [
  { name: "filesystem", category: "Endpoint Skills", description: "fs", url: "https://x/fs" },
  { name: "github", category: "Endpoint Skills", description: "gh", url: "https://x/gh" },
  { name: "weather", category: "Utility Skills", description: "w", url: "https://x/w" },
];

describe("groupSkillsByCategory", () => {
  it("groups skills under their category", () => {
    const grouped = groupSkillsByCategory(sample);
    expect(Object.keys(grouped).sort()).toEqual([
      "Endpoint Skills",
      "Utility Skills",
    ]);
    expect(grouped["Endpoint Skills"].map((s) => s.name)).toEqual([
      "filesystem",
      "github",
    ]);
    expect(grouped["Utility Skills"].map((s) => s.name)).toEqual(["weather"]);
  });

  it("returns an empty grouping for empty input", () => {
    expect(groupSkillsByCategory([])).toEqual({});
  });

  it("is pure — does not mutate the input array", () => {
    const input = sample.slice();
    const before = JSON.stringify(input);
    groupSkillsByCategory(input);
    expect(JSON.stringify(input)).toBe(before);
  });

  it("is deterministic — repeated calls give equal results", () => {
    expect(groupSkillsByCategory(sample)).toEqual(groupSkillsByCategory(sample));
  });

  it("buckets a single uncategorized-like category correctly", () => {
    const one: Skill[] = [
      { name: "solo", category: "Misc", description: "d", url: "u" },
    ];
    expect(groupSkillsByCategory(one)).toEqual({ Misc: one });
  });
});
