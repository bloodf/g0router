import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/skills")({
  component: () => <ComingSoon title="Skills" icon="extension" />,
});
