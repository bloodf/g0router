import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/mitm")({
  component: MitmPage,
});

function MitmPage() {
  return <h1>MITM</h1>;
}
