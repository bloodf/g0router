import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/proxy-pools")({
  component: ProxyPoolsPage,
});

function ProxyPoolsPage() {
  return <h1>Proxy Pools</h1>;
}
