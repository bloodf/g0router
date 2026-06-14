import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ProviderIcon } from "@/components/ui/provider-icon";

export const Route = createFileRoute("/traffic")({
  component: TrafficPage,
});

interface TrafficEvent {
  timestamp: string;
  key_id?: string;
  provider: string;
  model: string;
  status_class?: string;
  status_code?: number;
  latency_ms?: number;
}

const MAX_ROWS = 50;

function TrafficPage() {
  const [events, setEvents] = React.useState<TrafficEvent[]>([]);

  // Live traffic feed via EventSource("/api/traffic/stream") — the mock+fixture
  // surface (handlers/streams.ts + fixture.ts MockEventSource pushes rows).
  React.useEffect(() => {
    if (typeof EventSource === "undefined") return;
    const es = new EventSource("/api/traffic/stream");
    // addEventListener (not .onmessage=) so the e2e MockEventSource (fixture.ts)
    // synthetic dispatchEvent frames are received.
    const onMessage = (ev: MessageEvent) => {
      try {
        const data = JSON.parse(ev.data) as TrafficEvent;
        setEvents((prev) => [data, ...prev].slice(0, MAX_ROWS));
      } catch {
        // ignore malformed frames
      }
    };
    es.addEventListener("message", onMessage as EventListener);
    return () => {
      es.removeEventListener("message", onMessage as EventListener);
      es.close();
    };
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Traffic</h1>
      <Card padding="none">
        <table className="w-full text-sm" data-testid="traffic-table">
          <thead>
            <tr className="border-b border-border text-left text-muted-foreground">
              <th className="px-4 py-2 font-medium">Time</th>
              <th className="px-4 py-2 font-medium">Provider</th>
              <th className="px-4 py-2 font-medium">Model</th>
              <th className="px-4 py-2 font-medium">Status</th>
              <th className="px-4 py-2 font-medium">Latency</th>
            </tr>
          </thead>
          <tbody>
            {events.map((ev, i) => (
              <tr key={`${ev.timestamp}-${i}`} data-testid="traffic-row" className="border-b border-border/50">
                <td className="whitespace-nowrap px-4 py-2 text-muted-foreground">
                  {new Date(ev.timestamp).toLocaleTimeString("en-US")}
                </td>
                <td className="px-4 py-2">
                  <span className="inline-flex items-center gap-2">
                    <ProviderIcon slug={ev.provider.toLowerCase()} name={ev.provider} size="sm" />
                    {ev.provider}
                  </span>
                </td>
                <td className="px-4 py-2 text-foreground">{ev.model}</td>
                <td className="px-4 py-2">
                  <Badge variant={ev.status_class === "2xx" ? "success" : "error"}>
                    {ev.status_code ?? ev.status_class ?? "-"}
                  </Badge>
                </td>
                <td className="px-4 py-2">{ev.latency_ms !== undefined ? `${ev.latency_ms}ms` : "-"}</td>
              </tr>
            ))}
            {events.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-6 text-center text-muted-foreground">
                  Waiting for live traffic…
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
