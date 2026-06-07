import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/prompts")({
  component: () => <ComingSoon title="Prompts" icon="article" />,
});
