import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/endpoint")({
  component: EndpointPage,
});

function EndpointPage() {
  return <h1>Endpoint</h1>;
}
