import * as React from "react";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { apiFetch } from "@/lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import type { UsagePeriod } from "./usage-stats";

// UsageCharts renders recharts line charts from GET /api/usage/chart?period=
// (internal/admin/usage.go:120). The mock body serves a columnar shape
// {buckets, tokens_input, tokens_output, costs, requests}; the real Go returns
// []Bucket. The component is tolerant of both — it flattens to one row per
// bucket regardless of the source shape.
interface ColumnarChart {
  buckets: string[];
  tokens_input: number[];
  tokens_output: number[];
  costs: number[];
  requests: number[];
}
interface BucketRow {
  label?: string;
  requests: number;
  prompt_tokens: number;
  completion_tokens: number;
  cost: number;
}
type ChartResponse = ColumnarChart | BucketRow[];

interface ChartPoint {
  label: string;
  tokens: number;
  requests: number;
  cost: number;
}

function toPoints(data: ChartResponse): ChartPoint[] {
  if (Array.isArray(data)) {
    return data.map((b, i) => ({
      label: b.label || String(i),
      tokens: (b.prompt_tokens ?? 0) + (b.completion_tokens ?? 0),
      requests: b.requests ?? 0,
      cost: b.cost ?? 0,
    }));
  }
  const buckets = data?.buckets ?? [];
  return buckets.map((label, i) => ({
    label,
    tokens: (data.tokens_input?.[i] ?? 0) + (data.tokens_output?.[i] ?? 0),
    requests: data.requests?.[i] ?? 0,
    cost: data.costs?.[i] ?? 0,
  }));
}

export interface UsageChartsProps {
  period?: UsagePeriod;
}

export function UsageCharts({ period = "7d" }: UsageChartsProps) {
  const [points, setPoints] = React.useState<ChartPoint[]>([]);
  // chart has no "all" period (usage.go:124); coerce to a valid chart period.
  const chartPeriod = period === "all" ? "7d" : period;

  React.useEffect(() => {
    let cancelled = false;
    apiFetch<ChartResponse>(`/api/usage/chart?period=${chartPeriod}`)
      .then((data) => {
        if (!cancelled) setPoints(toPoints(data));
      })
      .catch(() => {
        if (!cancelled) setPoints([]);
      });
    return () => {
      cancelled = true;
    };
  }, [chartPeriod]);

  return (
    <Card padding="md" data-testid="usage-charts">
      <CardHeader>
        <CardTitle>Requests over time</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-64 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={points}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
              <XAxis dataKey="label" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip />
              <Line type="monotone" dataKey="requests" stroke="#6366f1" dot={false} />
              <Line type="monotone" dataKey="tokens" stroke="#10b981" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
