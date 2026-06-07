import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/proxy-pools")({
  component: () => <ComingSoon title="Proxy Pools" icon="lan" />,
});
