import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import type { PromptTemplate } from "@/lib/types";

export const Route = createFileRoute("/_app/prompts")({
  component: () => (
    <CrudPage<PromptTemplate>
      title="Prompt Templates"
      description="System prompts applied to matching models."
      icon="article"
      endpoint="/api/prompt-templates"
      queryKey={["prompt-templates"]}
      emptyTitle="No prompt templates"
      emptyDescription="Create templates that attach system prompts to selected models."
      fields={[
        { name: "name", label: "Name", required: true },
        { name: "system_prompt", label: "System prompt", type: "textarea", required: true },
        { name: "models", label: "Models (comma-separated)", type: "textarea" },
        { name: "is_active", label: "Active", type: "switch" },
      ]}
      initialValues={(row) => ({
        name: row?.name ?? "",
        system_prompt: row?.system_prompt ?? "",
        models: row?.models?.join(", ") ?? "",
        is_active: row?.is_active ?? true,
      })}
      transformBody={(values) => ({
        ...values,
        models: values.models
          ? String(values.models)
              .split(",")
              .map((s: string) => s.trim())
              .filter(Boolean)
          : [],
      })}
      columns={[
        { header: "Name", accessorKey: "name" },
        {
          header: "Models",
          cell: ({ row }) => row.original.models?.length ?? 0,
        },
        {
          header: "Status",
          cell: ({ row }) => (
            <StatusBadge variant={row.original.is_active ? "success" : "muted"} dot>
              {row.original.is_active ? "active" : "inactive"}
            </StatusBadge>
          ),
        },
      ]}
    />
  ),
});
