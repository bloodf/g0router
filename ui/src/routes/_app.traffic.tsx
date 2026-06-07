import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { PageHeader } from "@/components/common/PageHeader";
import { ProviderTopology } from "@/components/topology/ProviderTopology";
import { TrafficSummary } from "@/components/topology/TrafficSummary";

export const Route = createFileRoute("/_app/traffic")({
  component: TrafficPage,
});

function TrafficPage() {
  const [paused, setPaused] = useState(false);
  return (
    <div>
      <PageHeader
        title="Traffic"
        description="Live topology of API keys → combos → providers. Animated edges show traffic in the selected window."
        icon="graph_3"
      />
      <TrafficSummary paused={paused} onPausedChange={setPaused} />
      <ProviderTopology variant="full" paused={paused} onPausedChange={setPaused} />
    </div>
  );
}
