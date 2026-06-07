import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/guardrails")({
  component: () => <ComingSoon title="Guardrails" icon="shield" />,
});
