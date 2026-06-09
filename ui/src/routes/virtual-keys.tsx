import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/virtual-keys")({
  component: VirtualKeysPage,
});

function VirtualKeysPage() {
  return <h1>Virtual Keys</h1>;
}
