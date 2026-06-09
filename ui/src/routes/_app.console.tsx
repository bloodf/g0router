import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useConsoleStream } from "@/lib/hooks/useConsoleStream";
import { PageHeader } from "@/components/common/PageHeader";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/common/Icon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { format } from "date-fns";

function ConsolePage() {
  const { logs, clear, status } = useConsoleStream({});
  const [filter, setFilter] = useState<string>("ALL");
  const filtered = logs.filter((l) => filter === "ALL" || l.level === filter);
  const colors: Record<string, string> = {
    LOG: "text-success",
    INFO: "text-info",
    WARN: "text-warning",
    ERROR: "text-destructive",
    DEBUG: "text-brand",
  };

  const statusVariant: Record<string, "success" | "warning" | "danger" | "muted"> = {
    open: "success",
    connecting: "warning",
    error: "danger",
    closed: "muted",
  };

  return (
    <div>
      <PageHeader
        title="Console"
        description="Live server console stream."
        icon="terminal"
        actions={
          <>
            <StatusBadge variant={statusVariant[status] ?? "muted"} dot className="text-xs">
              {status === "open" ? "Live" : status}
            </StatusBadge>
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              aria-label="Log level filter"
              className="bg-surface-2 border border-border rounded-lg px-3 py-1.5 text-xs"
            >
              {["ALL", "LOG", "INFO", "WARN", "ERROR", "DEBUG"].map((l) => (
                <option key={l}>{l}</option>
              ))}
            </select>
            <Button variant="outline" onClick={clear}>
              <Icon name="clear_all" size={16} className="mr-1" /> Clear
            </Button>
          </>
        }
      />
      <Card className="card-elev border-border p-3 font-mono text-xs h-[calc(100vh-220px)] overflow-y-auto custom-scrollbar bg-surface">
        {filtered.map((l, i) => (
          <div key={`${l.timestamp}-${i}`} className="flex gap-2 py-0.5">
            <span className="text-text-muted">
              {format(new Date(l.timestamp), "HH:mm:ss")}
            </span>
            <span className={`${colors[l.level] || "text-foreground"} font-semibold w-12`}>
              {l.level}
            </span>
            <span className="text-foreground flex-1">{l.message}</span>
          </div>
        ))}
      </Card>
    </div>
  );
}

export const Route = createFileRoute("/_app/console")({
  component: ConsolePage,
});
