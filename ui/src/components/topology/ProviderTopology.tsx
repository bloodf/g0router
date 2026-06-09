import { useEffect, useMemo, useRef, useState } from "react";
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  type Edge,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import type {
  ApiKey,
  Combo,
  Connection,
  Provider,
  TrafficEvent,
} from "@/lib/types";
import { layoutFlow } from "./layout";
import { nodeTypes } from "./nodes";
import { useTrafficStream } from "@/lib/hooks/useTrafficStream";
import { Button } from "@/components/ui/button";
import { Icon } from "../common/Icon";
import { TopologyLegend } from "./TopologyLegend";
import {
  TopologyFilterBar,
  DEFAULT_FILTERS,
  type TopologyFilters,
} from "./TopologyFilters";
import { NodeDrawer, type SelectedNode } from "./NodeDrawer";

interface Props {
  variant?: "full" | "compact";
  providerFilter?: string;
  paused?: boolean;
  onPausedChange?: (paused: boolean) => void;
}

interface EdgeStat {
  count: number;
  errors: number;
  total_latency: number;
  last_ts: number;
}

const EMPTY_ARR: never[] = [];

function eventStatus(ev: TrafficEvent): "success" | "error" {
  return ev.status_class.startsWith("2") ? "success" : "error";
}

export function ProviderTopology({
  variant = "full",
  providerFilter,
  paused: pausedProp,
  onPausedChange,
}: Props) {
  const [pausedInternal, setPausedInternal] = useState(false);
  const paused = pausedProp ?? pausedInternal;
  const setPaused = (v: boolean) =>
    onPausedChange ? onPausedChange(v) : setPausedInternal(v);
  const [filters, setFilters] = useState<TopologyFilters>(DEFAULT_FILTERS);
  const [selected, setSelected] = useState<SelectedNode | null>(null);
  const [tick, setTick] = useState(0);

  const { data: providers = EMPTY_ARR } = useQuery({
    queryKey: ["providers"],
    queryFn: () => apiFetch<Provider[]>("/api/providers"),
  });
  const { data: connections = EMPTY_ARR } = useQuery({
    queryKey: ["connections"],
    queryFn: () => apiFetch<Connection[]>("/api/connections"),
  });
  const { data: combos = EMPTY_ARR } = useQuery({
    queryKey: ["combos"],
    queryFn: () => apiFetch<Combo[]>("/api/combos"),
  });
  const { data: keys = EMPTY_ARR } = useQuery({
    queryKey: ["keys"],
    queryFn: () => apiFetch<ApiKey[]>("/api/keys"),
  });

  const { events, lastEvent } = useTrafficStream({ enabled: !paused });

  // Buffer of recent events with the tagged edge ids for stat lookups.
  const eventLogRef = useRef<
    Array<{ ev: TrafficEvent; edges: string[]; ts: number }>
  >([]);

  useEffect(() => {
    if (!lastEvent) return;
    const keyId = `key-${lastEvent.key_id}`;
    const provId = `provider-${lastEvent.provider}`;

    // Find all combos that route to this provider; animate key→combo and combo→provider edges.
    const matchingCombos = combos.filter((c) =>
      c.steps?.some((s) => s.provider === lastEvent.provider),
    );

    const edges: string[] = [];
    for (const combo of matchingCombos) {
      edges.push(`${keyId}__combo-${combo.id}`);
      edges.push(`combo-${combo.id}__${provId}`);
    }

    eventLogRef.current.unshift({
      ev: lastEvent,
      edges,
      ts: Date.now(),
    });
    // cap buffer
    if (eventLogRef.current.length > 500) {
      eventLogRef.current = eventLogRef.current.slice(0, 500);
    }
  }, [lastEvent, combos]);

  // Drive recompute of edge stats so badges/animation fade naturally.
  useEffect(() => {
    const t = setInterval(() => setTick((x) => x + 1), 1000);
    return () => clearInterval(t);
  }, []);

  // Filter providers per UI filters.
  const visibleProviders = useMemo(() => {
    return providers.filter((p) => {
      if (providerFilter && p.id !== providerFilter) return false;
      if (p.connection_count === 0) return false;
      if (filters.status !== "all" && p.status !== filters.status) return false;
      if (filters.auth_type !== "all") {
        const connsForProv = connections.filter((c) => c.provider === p.id);
        const hasAuth = connsForProv.some((c) => c.auth_type === filters.auth_type);
        if (!hasAuth) return false;
      }
      return true;
    });
  }, [providers, connections, providerFilter, filters]);

  // Compute per-edge stats within the time window.
  const { edgeStats, activeEdgeIds } = useMemo(() => {
    const windowMs = filters.window_sec * 1000;
    const now = Date.now();
    const stats = new Map<string, EdgeStat>();
    const active = new Set<string>();
    for (const entry of eventLogRef.current) {
      if (now - entry.ts > windowMs) break;
      for (const eid of entry.edges) {
        const s = stats.get(eid) ?? {
          count: 0,
          errors: 0,
          total_latency: 0,
          last_ts: 0,
        };
        s.count += 1;
        if (eventStatus(entry.ev) === "error") s.errors += 1;
        s.total_latency += entry.ev.latency_ms;
        s.last_ts = Math.max(s.last_ts, entry.ts);
        stats.set(eid, s);
      }
      // Recently animated (last 1.8s) — for dashed traffic styling.
      if (now - entry.ts < 1800) {
        for (const eid of entry.edges) active.add(eid);
      }
    }
    return { edgeStats: stats, activeEdgeIds: active };
    // tick/lastEvent drive recompute because the source is a mutable ref.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, lastEvent, filters.window_sec]);

  // Build graph
  const { nodes, edges } = useMemo(() => {
    const ns: Node[] = [];
    const es: Edge[] = [];

    const activeKeys = keys.filter((k) => k.is_active);
    const activeCombos = combos.filter((c) => c.is_active);

    activeKeys.forEach((k) => {
      ns.push({
        id: `key-${k.id}`,
        type: "key",
        data: { label: k.name, prefix: k.prefix },
        position: { x: 0, y: 0 },
      });
    });

    activeCombos.forEach((c) => {
      ns.push({
        id: `combo-${c.id}`,
        type: "combo",
        data: { label: c.name, strategy: c.strategy },
        position: { x: 0, y: 0 },
      });
    });

    visibleProviders.forEach((p) => {
      ns.push({
        id: `provider-${p.id}`,
        type: "provider",
        data: {
          label: p.display_name,
          provider: p.id,
          status: p.status,
          connectionCount: p.connection_count,
        },
        position: { x: 0, y: 0 },
      });
    });

    const visibleProviderIds = new Set(visibleProviders.map((p) => p.id));

    const makeLabel = (stat?: EdgeStat) => {
      if (!stat || stat.count === 0) return undefined;
      const avg = Math.round(stat.total_latency / stat.count);
      return `${stat.count} · ${avg}ms${stat.errors > 0 ? ` · ⚠ ${stat.errors}` : ""}`;
    };
    const labelStyle = (stat?: EdgeStat): React.CSSProperties => ({
      fontSize: 10,
      fontWeight: 600,
      fill: stat && stat.errors > 0 ? "var(--danger)" : "var(--foreground)",
    });
    const labelBgStyle = (stat?: EdgeStat): React.CSSProperties => ({
      fill: stat && stat.errors > 0 ? "color-mix(in oklab, var(--danger) 12%, var(--surface))" : "var(--surface)",
      stroke: "var(--border)",
      strokeWidth: 1,
    });

    // key → combo
    activeKeys.forEach((k) => {
      activeCombos.forEach((c) => {
        const eid = `key-${k.id}__combo-${c.id}`;
        const stat = edgeStats.get(eid);
        es.push({
          id: eid,
          source: `key-${k.id}`,
          target: `combo-${c.id}`,
          className: activeEdgeIds.has(eid) ? "traffic-active" : "traffic-idle",
          label: makeLabel(stat),
          labelStyle: labelStyle(stat),
          labelBgStyle: labelBgStyle(stat),
          labelBgPadding: [4, 2],
          labelBgBorderRadius: 4,
        });
      });
    });

    // combo → provider (only steps that point at visible providers)
    activeCombos.forEach((c) => {
      const targets = new Set(c.steps?.map((s) => s.provider) ?? []);
      visibleProviders.forEach((p) => {
        if (!targets.has(p.id)) return;
        const eid = `combo-${c.id}__provider-${p.id}`;
        const stat = edgeStats.get(eid);
        es.push({
          id: eid,
          source: `combo-${c.id}`,
          target: `provider-${p.id}`,
          className: activeEdgeIds.has(eid) ? "traffic-active" : "traffic-idle",
          label: makeLabel(stat),
          labelStyle: labelStyle(stat),
          labelBgStyle: labelBgStyle(stat),
          labelBgPadding: [4, 2],
          labelBgBorderRadius: 4,
        });
      });
    });

    // Hide edges whose endpoints are filtered out.
    const nodeIds = new Set(ns.map((n) => n.id));
    const filteredEdges = es.filter(
      (e) => nodeIds.has(e.source) && nodeIds.has(e.target),
    );

    // Drop combos that have no outgoing visible provider edge.
    const usedNodeIds = new Set<string>();
    filteredEdges.forEach((e) => {
      usedNodeIds.add(e.source);
      usedNodeIds.add(e.target);
    });
    const trimmedNodes = ns.filter((n) => {
      if (n.id?.startsWith("provider-")) {
        return visibleProviderIds.has(n.id.replace("provider-", ""));
      }
      // keep keys/combos only if they participate in at least one edge
      return usedNodeIds.has(n.id);
    });

    return { nodes: layoutFlow(trimmedNodes, filteredEdges, "LR"), edges: filteredEdges };
  }, [visibleProviders, combos, keys, edgeStats, activeEdgeIds]);

  const h = variant === "compact" ? 320 : "calc(100vh - 200px)";

  return (
    <div
      className="relative rounded-xl border border-border bg-surface overflow-hidden"
      style={{ height: h }}
    >
      {variant === "full" && (
        <>
          <div className="absolute top-3 left-3 z-10 flex items-center gap-1 bg-surface/90 backdrop-blur-md border border-border rounded-lg p-1 shadow-elev">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setPaused(!paused)}
              className="gap-1.5"
            >
              <Icon name={paused ? "play_arrow" : "pause"} size={16} />
              {paused ? "Resume" : "Pause"}
            </Button>
            <div className="h-5 w-px bg-border" />
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                eventLogRef.current = [];
                setTick((x) => x + 1);
              }}
              className="gap-1.5"
            >
              <Icon name="restart_alt" size={16} />
              Clear
            </Button>
          </div>
          <TopologyFilterBar value={filters} onChange={setFilters} />
          <TopologyLegend window_sec={filters.window_sec} />
        </>
      )}
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        proOptions={{ hideAttribution: false }}
        nodesDraggable={variant === "full"}
        zoomOnScroll={variant === "full"}
        panOnDrag={variant === "full"}
        onNodeClick={(_, n) => {
          const [kind, ...rest] = n.id.split("-");
          setSelected({
            kind: kind as SelectedNode["kind"],
            id: rest.join("-"),
          });
        }}
      >
        <Background gap={20} size={1} />
        {variant === "full" && <Controls />}
        {variant === "full" && <MiniMap pannable zoomable />}
      </ReactFlow>
      <NodeDrawer
        selected={selected}
        onClose={() => setSelected(null)}
        events={events}
      />
    </div>
  );
}
