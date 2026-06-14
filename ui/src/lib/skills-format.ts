import type { Skill } from "./types";

// groupSkillsByCategory buckets skills under their `category` while preserving
// the input order within each category. Pure: it copies into fresh arrays and
// never mutates the source. This is the authoritative skills-grouping proof
// (§1.3 point 3).
export function groupSkillsByCategory(
  skills: Skill[]
): Record<string, Skill[]> {
  const grouped: Record<string, Skill[]> = {};
  for (const skill of skills) {
    if (!grouped[skill.category]) {
      grouped[skill.category] = [];
    }
    grouped[skill.category].push(skill);
  }
  return grouped;
}
