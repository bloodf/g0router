import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { CardSkeleton } from "@/components/ui/skeleton";
import { groupSkillsByCategory } from "@/lib/skills-format";
import { useNotificationStore } from "@/stores/notification";
import type { Skill } from "@/lib/types";

export const Route = createFileRoute("/skills")({
  component: SkillsPage,
});

// SkillsPage (PAR-UI-020, NEW route §1.7) reads GET /api/skills, groups skills by
// category, and renders each as a row with name/description/url plus a
// copy-to-clipboard button (navigator.clipboard.writeText, transient "Copied!").
// Variant-HAVE against the registered /api/skills mock; no Go backend yet
// (§8 ESC-1c). Adding this file regenerates routeTree.gen.ts to register /skills.
function SkillsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [skills, setSkills] = React.useState<Skill[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [copied, setCopied] = React.useState<string | null>(null);

  React.useEffect(() => {
    apiFetch<Skill[]>("/api/skills")
      .then((rows) => {
        setSkills(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setSkills([]);
        setLoading(false);
        pushToast({ message: "Failed to load skills" });
      });
  }, [pushToast]);

  async function copy(skill: Skill) {
    try {
      await navigator.clipboard.writeText(skill.url);
      setCopied(skill.name);
      window.setTimeout(() => setCopied(null), 1500);
    } catch {
      pushToast({ message: "Failed to copy to clipboard" });
    }
  }

  const grouped = groupSkillsByCategory(skills);

  return (
    <div className="flex flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-foreground">Skills</h1>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : skills.length === 0 ? (
        <p className="text-sm text-muted-foreground">No skills available.</p>
      ) : (
        Object.entries(grouped).map(([category, items]) => (
          <section key={category} className="flex flex-col gap-2">
            <h2 className="text-sm font-semibold text-foreground">{category}</h2>
            {items.map((skill) => (
              <div
                key={skill.name}
                data-testid="skill-row"
                className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
              >
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {skill.name}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {skill.description}
                  </p>
                  <p className="text-xs text-muted-foreground">{skill.url}</p>
                </div>
                <Button
                  data-testid="skill-copy"
                  variant="ghost"
                  size="sm"
                  onClick={() => copy(skill)}
                  aria-label={`Copy ${skill.name} URL`}
                >
                  <span className="material-symbols-outlined text-base">
                    {copied === skill.name ? "check" : "content_copy"}
                  </span>
                  {copied === skill.name ? "Copied!" : "Copy"}
                </Button>
              </div>
            ))}
          </section>
        ))
      )}
    </div>
  );
}
