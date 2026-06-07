import dagre from "dagre";
import type { Edge, Node } from "@xyflow/react";

const W = 220;
const H = 76;

export function layoutFlow(
  nodes: Node[],
  edges: Edge[],
  direction: "LR" | "TB" = "LR",
) {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  // Roomier spacing so providers don't crowd into a single column with no air.
  g.setGraph({
    rankdir: direction,
    nodesep: 36,
    ranksep: 140,
    edgesep: 24,
    marginx: 24,
    marginy: 24,
    align: "UL",
  });

  nodes.forEach((n) => g.setNode(n.id, { width: W, height: H }));
  edges.forEach((e) => g.setEdge(e.source, e.target));
  dagre.layout(g);

  return nodes.map((n) => {
    const p = g.node(n.id);
    return {
      ...n,
      targetPosition: direction === "LR" ? "left" : "top",
      sourcePosition: direction === "LR" ? "right" : "bottom",
      position: { x: p.x - W / 2, y: p.y - H / 2 },
    } as Node;
  });
}
