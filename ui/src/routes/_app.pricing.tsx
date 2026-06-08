import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import type { PricingOverride } from "@/lib/types";

export const Route = createFileRoute("/_app/pricing")({
  component: () => (
    <CrudPage<PricingOverride>
      title="Pricing"
      description="Per-model pricing overrides ($ per million tokens)."
      icon="sell"
      endpoint="/api/pricing"
      queryKey={["pricing"]}
      emptyTitle="No pricing overrides"
      emptyDescription="Add an override to bill a specific model at custom $/M input and output rates."
      fields={[
        { name: "provider", label: "Provider", required: true },
        { name: "model", label: "Model", required: true },
        { name: "input_cost", label: "Input ($/M)", type: "number", required: true },
        { name: "output_cost", label: "Output ($/M)", type: "number", required: true },
      ]}
      columns={[
        { header: "Provider", accessorKey: "provider" },
        {
          header: "Model",
          cell: ({ row }) => <code className="text-xs">{row.original.model}</code>,
        },
        {
          header: "Input",
          cell: ({ row }) => `$${row.original.input_cost}/M`,
        },
        {
          header: "Output",
          cell: ({ row }) => `$${row.original.output_cost}/M`,
        },
      ]}
    />
  ),
});
