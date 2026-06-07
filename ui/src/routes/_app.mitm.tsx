import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/mitm")({
  component: () => <ComingSoon title="MITM" icon="security" />,
});
