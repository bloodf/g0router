import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/PageHeader";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Icon } from "@/components/common/Icon";
import { EmptyState } from "@/components/common/EmptyState";
import { CardsGridSkeleton, ErrorState } from "@/components/common/Skeletons";

export const Route = createFileRoute("/_app/skills")({
  component: SkillsPage,
});

type Skill = {
  name: string;
  category: string;
  description: string;
  url: string;
};

function SkillsPage() {
  const { data: skills = [], isLoading, isError, error, refetch } = useQuery<Skill[]>({
    queryKey: ["skills"],
    queryFn: () => apiFetch("/api/skills"),
  });

  return (
    <div>
      <PageHeader
        title="Skills"
        description="Discover agent skills exposed by this gateway."
        icon="extension"
        actions={
          <Button variant="outline" onClick={() => refetch()}>
            <Icon name="refresh" size={16} className="mr-1.5" />
            Refresh
          </Button>
        }
      />

      {isLoading ? (
        <CardsGridSkeleton count={8} height="h-36" />
      ) : isError ? (
        <ErrorState
          title="Couldn’t load skills"
          error={error}
          onRetry={() => refetch()}
        />
      ) : skills.length === 0 ? (
        <EmptyState
          icon="extension"
          title="No skills available"
          description="There are no skills registered on this gateway yet."
        />
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          {skills.map((skill) => (
            <SkillCard key={skill.name} skill={skill} />
          ))}
        </div>
      )}
    </div>
  );
}

function SkillCard({ skill }: { skill: Skill }) {
  return (
    <Card className="p-4 card-elev border-border h-full flex flex-col">
      <div className="flex items-start justify-between gap-3">
        <h3 className="font-semibold leading-tight min-w-0">{skill.name}</h3>
        <StatusBadge variant="primary">{skill.category}</StatusBadge>
      </div>
      <p className="text-sm text-text-muted mt-2 flex-1 line-clamp-3">
        {skill.description}
      </p>
      <a
        href={skill.url}
        target="_blank"
        rel="noreferrer"
        className="inline-flex items-center gap-1.5 text-sm font-medium text-brand-600 hover:text-brand-700 mt-4 pt-3 border-t border-border"
      >
        <Icon name="open_in_new" size={16} />
        Open skill
      </a>
    </Card>
  );
}
