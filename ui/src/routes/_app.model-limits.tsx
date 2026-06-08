import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import type { ModelLimit } from "@/lib/types";

export const Route = createFileRoute("/_app/model-limits")({
  component: () => (
    <CrudPage<ModelLimit>
      title="Model Limits"
      description="Per-model token and rate constraints."
      icon="speed"
      endpoint="/api/model-limits"
      queryKey={["model-limits"]}
      emptyTitle="No model limits"
      emptyDescription="Define max tokens and RPM restrictions per model."
      fields={[
        { name: "model", label: "Model", required: true },
        { name: "max_tokens", label: "Max tokens", type: "number" },
        { name: "max_rpm", label: "Max RPM", type: "number" },
        { name: "allowed_key_ids", label: "Allowed key IDs (comma-separated)", type: "textarea" },
      ]}
      initialValues={(row) => ({
        model: row?.model ?? "",
        max_tokens: row?.max_tokens ?? "",
        max_rpm: row?.max_rpm ?? "",
        allowed_key_ids: row?.allowed_key_ids?.join(", ") ?? "",
      })}
      transformBody={(values) => ({
        ...values,
        max_tokens: values.max_tokens ? Number(values.max_tokens) : null,
        max_rpm: values.max_rpm ? Number(values.max_rpm) : null,
        allowed_key_ids: values.allowed_key_ids
          ? String(values.allowed_key_ids)
              .split(",")
              .map((s: string) => s.trim())
              .filter(Boolean)
          : [],
      })}
      columns={[
        { header: "Model", accessorKey: "model" },
        {
          header: "Max tokens",
          cell: ({ row }) => row.original.max_tokens ?? "—",
        },
        {
          header: "Max RPM",
          cell: ({ row }) => row.original.max_rpm ?? "—",
        },
        {
          header: "Allowed keys",
          cell: ({ row }) => row.original.allowed_key_ids?.length ?? 0,
        },
      ]}
    />
  ),
});
