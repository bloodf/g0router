import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/feature-flags")({
  component: FeatureFlagsPage,
});

function FeatureFlagsPage() {
  return <h1>Feature Flags</h1>;
}
