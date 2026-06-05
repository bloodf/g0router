import { useEffect, useRef, useState } from "react";
import { ApiError, streamTraffic, type TrafficEvent } from "../api";
import { ErrorState, Panel } from "../components/Primitives";

const ROLLING_WINDOW_MS = 30_000;
const MAX_EVENTS = 200;

type NodeKind = "key" | "provider";

type GraphNode = {
  id: string;
  label: string;
  kind: NodeKind;
};

type EdgeKey = string; // `${keyId}::${provider}`

type EdgeState = {
  keyId: string;
  provider: string;
  count: number;
  lastAt: number; // epoch ms
  pulseAt: number; // epoch ms of most recent event for animation
};

type TrafficState = {
  nodes: GraphNode[];
  edges: Map<EdgeKey, EdgeState>;
  recentEvents: TrafficEvent[];
};

type StreamStatus = "connecting" | "connected" | "error" | "auth-expired";

function edgeKey(keyId: string, provider: string): EdgeKey {
  return `${keyId}::${provider}`;
}

function normalizeKeyId(raw: string): string {
  return raw.trim() === "" ? "anonymous" : raw;
}

function applyEvent(prev: TrafficState, event: TrafficEvent, now: number): TrafficState {
  const keyLabel = normalizeKeyId(event.key_id);
  const provider = event.provider;

  // Nodes — build new set immutably.
  const existingIds = new Set(prev.nodes.map((n) => `${n.kind}:${n.id}`));
  const newNodes = [...prev.nodes];
  if (!existingIds.has(`key:${keyLabel}`)) {
    newNodes.push({ id: keyLabel, label: keyLabel, kind: "key" });
  }
  if (!existingIds.has(`provider:${provider}`)) {
    newNodes.push({ id: provider, label: provider, kind: "provider" });
  }

  // Edges.
  const key = edgeKey(keyLabel, provider);
  const existing = prev.edges.get(key);
  const updatedEdge: EdgeState = {
    keyId: keyLabel,
    provider,
    count: (existing?.count ?? 0) + 1,
    lastAt: now,
    pulseAt: now
  };
  const newEdges = new Map(prev.edges);
  newEdges.set(key, updatedEdge);

  // Rolling window.
  const cutoff = now - ROLLING_WINDOW_MS;
  const trimmed = [...prev.recentEvents, event].slice(-MAX_EVENTS).filter(
    (e) => new Date(e.timestamp).getTime() >= cutoff
  );

  return { nodes: newNodes, edges: newEdges, recentEvents: trimmed };
}

// Layout constants.
const SVG_W = 700;
const SVG_H = 340;
const GW_CX = SVG_W / 2;
const GW_CY = SVG_H / 2;
const GW_R = 32;

const KEY_COL_X = 100;
const PROVIDER_COL_X = SVG_W - 100;
const NODE_R = 22;

function columnY(index: number, total: number): number {
  const spacing = Math.min(60, (SVG_H - 60) / Math.max(total, 1));
  const totalHeight = (total - 1) * spacing;
  return SVG_H / 2 - totalHeight / 2 + index * spacing;
}

function isRecent(pulseAt: number, now: number): boolean {
  return now - pulseAt < 1500;
}

function edgeStrokeWidth(count: number): number {
  return Math.min(1 + Math.log2(count + 1), 4);
}

type PulseCircleProps = {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  active: boolean;
};

function PulseCircle({ x1, y1, x2, y2, active }: PulseCircleProps) {
  if (!active) return null;
  // Midpoint of the segment.
  const mx = (x1 + x2) / 2;
  const my = (y1 + y2) / 2;
  return (
    <circle cx={mx} cy={my} r={5} fill="#3b82f6">
      <animate attributeName="opacity" values="1;0" dur="1.4s" begin="0s" fill="freeze" />
      <animate attributeName="r" values="5;10" dur="1.4s" begin="0s" fill="freeze" />
    </circle>
  );
}

function TopologyGraph({
  graphState,
  now
}: {
  graphState: TrafficState;
  now: number;
}) {
  const keyNodes = graphState.nodes.filter((n) => n.kind === "key");
  const providerNodes = graphState.nodes.filter((n) => n.kind === "provider");

  return (
    <svg
      aria-label="Traffic topology"
      viewBox={`0 0 ${SVG_W} ${SVG_H}`}
      className="w-full max-w-2xl"
      style={{ minHeight: 240 }}
    >
      {/* Draw key→gateway edges */}
      {keyNodes.map((kn, i) => {
        const ky = columnY(i, keyNodes.length);
        const edge = graphState.edges.get(edgeKey(kn.id, "")) ?? null;
        const active = Array.from(graphState.edges.values()).some(
          (e) => e.keyId === kn.id && isRecent(e.pulseAt, now)
        );
        const totalCount = Array.from(graphState.edges.values())
          .filter((e) => e.keyId === kn.id)
          .reduce((s, e) => s + e.count, 0);
        const sw = edgeStrokeWidth(totalCount);
        return (
          <g key={`ke-${kn.id}`}>
            <line
              x1={KEY_COL_X + NODE_R}
              y1={ky}
              x2={GW_CX - GW_R}
              y2={GW_CY}
              stroke={active ? "#3b82f6" : "#d4d4d8"}
              strokeWidth={sw}
              strokeLinecap="round"
            />
            <PulseCircle
              x1={KEY_COL_X + NODE_R}
              y1={ky}
              x2={GW_CX - GW_R}
              y2={GW_CY}
              active={active}
            />
            {/* dummy ref for edge count */}
            {edge ? null : null}
          </g>
        );
      })}

      {/* Draw gateway→provider edges */}
      {providerNodes.map((pn, i) => {
        const py = columnY(i, providerNodes.length);
        const active = Array.from(graphState.edges.values()).some(
          (e) => e.provider === pn.id && isRecent(e.pulseAt, now)
        );
        const totalCount = Array.from(graphState.edges.values())
          .filter((e) => e.provider === pn.id)
          .reduce((s, e) => s + e.count, 0);
        const sw = edgeStrokeWidth(totalCount);
        return (
          <g key={`pe-${pn.id}`}>
            <line
              x1={GW_CX + GW_R}
              y1={GW_CY}
              x2={PROVIDER_COL_X - NODE_R}
              y2={py}
              stroke={active ? "#3b82f6" : "#d4d4d8"}
              strokeWidth={sw}
              strokeLinecap="round"
            />
            <PulseCircle
              x1={GW_CX + GW_R}
              y1={GW_CY}
              x2={PROVIDER_COL_X - NODE_R}
              y2={py}
              active={active}
            />
          </g>
        );
      })}

      {/* Key nodes */}
      {keyNodes.map((kn, i) => {
        const ky = columnY(i, keyNodes.length);
        const active = Array.from(graphState.edges.values()).some(
          (e) => e.keyId === kn.id && isRecent(e.pulseAt, now)
        );
        return (
          <g key={`kn-${kn.id}`}>
            <circle
              cx={KEY_COL_X}
              cy={ky}
              r={NODE_R}
              fill={active ? "#eff6ff" : "#f4f4f5"}
              stroke={active ? "#3b82f6" : "#a1a1aa"}
              strokeWidth={1.5}
            />
            <text
              x={KEY_COL_X}
              y={ky + 4}
              textAnchor="middle"
              fontSize={9}
              fill="#3f3f46"
              className="select-none"
            >
              {kn.label.length > 10 ? `${kn.label.slice(0, 9)}…` : kn.label}
            </text>
          </g>
        );
      })}

      {/* Gateway node */}
      <circle cx={GW_CX} cy={GW_CY} r={GW_R} fill="#18181b" />
      <text x={GW_CX} y={GW_CY + 5} textAnchor="middle" fontSize={12} fill="white" className="select-none font-semibold">
        gateway
      </text>

      {/* Provider nodes */}
      {providerNodes.map((pn, i) => {
        const py = columnY(i, providerNodes.length);
        const active = Array.from(graphState.edges.values()).some(
          (e) => e.provider === pn.id && isRecent(e.pulseAt, now)
        );
        return (
          <g key={`pn-${pn.id}`}>
            <circle
              cx={PROVIDER_COL_X}
              cy={py}
              r={NODE_R}
              fill={active ? "#eff6ff" : "#f4f4f5"}
              stroke={active ? "#3b82f6" : "#a1a1aa"}
              strokeWidth={1.5}
            />
            <text
              x={PROVIDER_COL_X}
              y={py + 4}
              textAnchor="middle"
              fontSize={9}
              fill="#3f3f46"
              className="select-none"
            >
              {pn.label.length > 10 ? `${pn.label.slice(0, 9)}…` : pn.label}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

const emptyGraph: TrafficState = {
  nodes: [],
  edges: new Map(),
  recentEvents: []
};

export function TrafficPage() {
  const [status, setStatus] = useState<StreamStatus>("connecting");
  const [error, setError] = useState<ApiError | null>(null);
  const [graphState, setGraphState] = useState<TrafficState>(emptyGraph);
  const [now, setNow] = useState(() => Date.now());
  const hasEvents = graphState.nodes.length > 0;

  // Tick to drive pulse animations.
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 200);
    return () => clearInterval(id);
  }, []);

  const retryRef = useRef(0);

  const connect = () => {
    setStatus("connecting");
    setError(null);
    setGraphState(emptyGraph);
    retryRef.current += 1;
  };

  useEffect(() => {
    const cleanup = streamTraffic(
      (event) => {
        setStatus("connected");
        setGraphState((prev) => applyEvent(prev, event, Date.now()));
      },
      (err) => {
        if (err.authExpired) {
          setStatus("auth-expired");
        } else {
          setStatus("error");
        }
        setError(err);
      }
    );
    return cleanup;
  }, [retryRef.current]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Panel
      title="Live traffic topology"
      description="Real-time key→gateway→provider flow from the SSE event stream."
    >
      {status === "error" && error ? (
        <ErrorState
          title="Could not connect to traffic stream"
          message={error.message}
          onRetry={connect}
        />
      ) : status === "auth-expired" && error ? (
        <ErrorState
          title="Session expired"
          message={error.message}
          onRetry={connect}
        />
      ) : (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <span
              className={`inline-block h-2 w-2 rounded-full ${
                status === "connected" ? "bg-emerald-500" : "bg-amber-400"
              }`}
            />
            <span className="text-xs text-zinc-500">
              {status === "connected" ? "Connected" : "Connecting…"}
            </span>
          </div>

          <TopologyGraph graphState={graphState} now={now} />

          {!hasEvents && (
            <p className="text-sm text-zinc-400">Waiting for live traffic…</p>
          )}

          {hasEvents && (
            <div className="mt-2 text-xs text-zinc-400">
              {graphState.recentEvents.length} event
              {graphState.recentEvents.length !== 1 ? "s" : ""} in rolling window ·{" "}
              {graphState.nodes.filter((n) => n.kind === "key").length} key
              {graphState.nodes.filter((n) => n.kind === "key").length !== 1 ? "s" : ""} ·{" "}
              {graphState.nodes.filter((n) => n.kind === "provider").length} provider
              {graphState.nodes.filter((n) => n.kind === "provider").length !== 1 ? "s" : ""}
            </div>
          )}
        </div>
      )}
    </Panel>
  );
}
