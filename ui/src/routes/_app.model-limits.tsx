import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/model-limits")({
  component: () => <ComingSoon title="Model Limits" icon="speed" />,
});
