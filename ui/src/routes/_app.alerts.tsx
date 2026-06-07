import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/alerts")({
  component: () => <ComingSoon title="Alert Channels" icon="notifications" />,
});
