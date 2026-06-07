import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/feature-flags")({
  component: () => <ComingSoon title="Feature Flags" icon="flag" />,
});
