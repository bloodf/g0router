import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import type { Alias } from "@/lib/mocks/types";

export const Route = createFileRoute("/_app/aliases")({
  component: () => (
    <CrudPage<Alias>
      title="Aliases"
      description="Short names that resolve to a specific provider/model."
      icon="label"
      endpoint="/api/aliases"
      queryKey={["aliases"]}
      emptyTitle="No aliases yet"
      emptyDescription="Add a short alias that resolves to a specific provider and model — e.g. fast → groq/llama-3.1-8b."
      fields={[
        { name: "alias", label: "Alias name", required: true },
        { name: "provider", label: "Provider", required: true },
        { name: "model", label: "Model", required: true },
      ]}
      columns={[
        {
          header: "Alias",
          accessorKey: "alias",
          cell: ({ row }) => (
            <code className="text-xs bg-brand-500/10 text-brand-600 px-2 py-0.5 rounded">
              {row.original.alias}
            </code>
          ),
        },
        { header: "Provider", accessorKey: "provider" },
        {
          header: "Model",
          accessorKey: "model",
          cell: ({ row }) => <code className="text-xs">{row.original.model}</code>,
        },
      ]}
    />
  ),
});
