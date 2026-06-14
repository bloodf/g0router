import * as React from "react";
import { ReactFlow, Background, type Node, type Edge } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { ModelStat, ProviderStat } from "./usage-stats";

// ProviderTopology renders a provider -> model graph (PAR-UI-047 topology) from
// the usage stats, using @xyflow/react. Ported from ref ProviderTopology.js,
// adapted to React 19 + the real Go stats shape.
export interface ProviderTopologyProps {
  byProvider: Record<string, ProviderStat>;
  byModel: Record<string, ModelStat>;
}

function buildGraph(
  byProvider: Record<string, ProviderStat>,
  byModel: Record<string, ModelStat>,
): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const providers = Object.keys(byProvider);

  providers.forEach((provider, pi) => {
    nodes.push({
      id: `p:${provider}`,
      position: { x: 0, y: pi * 90 },
      data: { label: provider },
      type: "input",
    });
  });

  Object.entries(byModel).forEach(([key, stat], mi) => {
    const provider = stat.provider || key.split("/")[0];
    const modelId = `m:${key}`;
    nodes.push({
      id: modelId,
      position: { x: 260, y: mi * 70 },
      data: { label: stat.raw_model || key },
    });
    if (byProvider[provider]) {
      edges.push({ id: `e:${provider}->${key}`, source: `p:${provider}`, target: modelId });
    }
  });

  return { nodes, edges };
}

export function ProviderTopology({ byProvider, byModel }: ProviderTopologyProps) {
  const { nodes, edges } = React.useMemo(
    () => buildGraph(byProvider, byModel),
    [byProvider, byModel],
  );

  if (nodes.length === 0) {
    return null;
  }

  return (
    <div data-testid="provider-topology" className="h-72 w-full rounded-xl border border-border bg-card">
      <ReactFlow nodes={nodes} edges={edges} fitView proOptions={{ hideAttribution: true }}>
        <Background />
      </ReactFlow>
    </div>
  );
}
