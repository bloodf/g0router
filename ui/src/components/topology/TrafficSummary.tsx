import { useEffect, useMemo, useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/common/Icon";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useTrafficStream } from "@/lib/hooks/useTrafficStream";
import type { TrafficEvent } from "@/lib/types";

export type TimeWindow = 30 | 120 | 300 | 900;
export type StatusFilter = "all" | "success" | "error";

interface Props {
  paused?: boolean;
  onPausedChange?: (paused: boolean) => void;
}

function eventStatus(ev: TrafficEvent): "success" | "error" {
  return ev.status_class.startsWith("2") ? "success" : "error";
}

export function TrafficSummary({ paused = false, onPausedChange }: Props) {
  const [windowSec, setWindowSec] = useState<TimeWindow>(120);
  const [status, setStatus] = useState<StatusFilter>("all");
  const [now, setNow] = useState<number>(Date.now);

  const { events, clear } = useTrafficStream({ enabled: !paused });

  // Drive recompute every second so counters tick down as events age out.
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, []);

  const stats = useMemo(() => {
    const cutoff = now - windowSec * 1000;
    const inWindow = events.filter(
      (ev) => new Date(ev.timestamp).getTime() >= cutoff && (status === "all" || eventStatus(ev) === status),
    );
    const errors = inWindow.filter((ev) => eventStatus(ev) === "error").length;
    const successes = inWindow.length - errors;
    const totalLatency = inWindow.reduce((s, ev) => s + ev.latency_ms, 0);
    const reqPerMin = inWindow.length / (windowSec / 60);
    const errorRate = inWindow.length ? (errors / inWindow.length) * 100 : 0;
    const avgLatency = inWindow.length ? totalLatency / inWindow.length : 0;

    const byProvider = new Map<string, number>();
    for (const ev of inWindow) {
      byProvider.set(ev.provider, (byProvider.get(ev.provider) ?? 0) + 1);
    }
    const top = Array.from(byProvider.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 4);

    return {
      total: inWindow.length,
      successes,
      errors,
      reqPerMin,
      errorRate,
      avgLatency,
      top,
    };
  }, [events, windowSec, status, now]);

  return (
    <Card className="card-elev border-border p-4 mb-4">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <div className="flex items-center gap-2">
          <Icon name="filter_alt" size={16} className="text-text-muted" />
          <span className="text-xs uppercase tracking-wider text-text-muted font-semibold">
            Filters
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Select
            value={String(windowSec)}
            onValueChange={(v) => setWindowSec(Number(v) as TimeWindow)}
          >
            <SelectTrigger className="h-8 w-[110px] text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="30">Last 30s</SelectItem>
              <SelectItem value="120">Last 2m</SelectItem>
              <SelectItem value="300">Last 5m</SelectItem>
              <SelectItem value="900">Last 15m</SelectItem>
            </SelectContent>
          </Select>
          <Select value={status} onValueChange={(v) => setStatus(v as StatusFilter)}>
            <SelectTrigger className="h-8 w-[130px] text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All status</SelectItem>
              <SelectItem value="success">Success only</SelectItem>
              <SelectItem value="error">Errors only</SelectItem>
            </SelectContent>
          </Select>
          {onPausedChange && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onPausedChange(!paused)}
              className="gap-1.5 h-8"
            >
              <Icon name={paused ? "play_arrow" : "pause"} size={14} />
              {paused ? "Resume" : "Pause"}
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => clear()}
            className="gap-1.5 h-8"
          >
            <Icon name="restart_alt" size={14} />
            Reset
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2">
        <Metric label="Requests" value={stats.total} icon="bolt" />
        <Metric
          label="Req / min"
          value={stats.reqPerMin.toFixed(1)}
          icon="trending_up"
          tone="info"
        />
        <Metric
          label="Avg latency"
          value={`${Math.round(stats.avgLatency)}ms`}
          icon="schedule"
        />
        <Metric
          label="Error rate"
          value={`${stats.errorRate.toFixed(1)}%`}
          icon="error"
          tone={stats.errorRate > 5 ? "danger" : stats.errorRate > 1 ? "warning" : "success"}
        />
      </div>

      {stats.top.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 mt-3 pt-3 border-t border-border">
          <span className="text-[11px] uppercase tracking-wider text-text-muted font-semibold">
            Top providers
          </span>
          {stats.top.map(([p, n]) => (
            <div
              key={p}
              className="flex items-center gap-1.5 px-2 py-1 rounded-md bg-surface-2 text-xs"
            >
              <ProviderIcon provider={p} size={16} />
              <span className="capitalize font-medium">{p}</span>
              <span className="text-text-muted tabular-nums">{n}</span>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

function Metric({
  label,
  value,
  icon,
  tone,
}: {
  label: string;
  value: string | number;
  icon: string;
  tone?: "info" | "success" | "warning" | "danger";
}) {
  const toneClass: Record<string, string> = {
    info: "text-info bg-info/10",
    success: "text-success bg-success/10",
    warning: "text-warning bg-warning/10",
    danger: "text-destructive bg-destructive/10",
  };
  return (
    <div className="flex items-center gap-2 p-2.5 rounded-lg bg-surface-2/60 border border-border">
      <div
        className={
          "w-8 h-8 rounded-md flex items-center justify-center flex-shrink-0 " +
          (tone ? toneClass[tone] : "bg-surface text-text-muted")
        }
      >
        <Icon name={icon} size={16} />
      </div>
      <div className="min-w-0">
        <div className="text-[10px] uppercase tracking-wider text-text-muted truncate">
          {label}
        </div>
        <div className="text-base font-semibold tabular-nums truncate">{value}</div>
      </div>
    </div>
  );
}
