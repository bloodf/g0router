import { createFileRoute } from "@tanstack/react-router";
import { Card } from "@/components/ui/card";
import { LiveConsoleLogViewer } from "@/components/console/console-log-viewer";

export const Route = createFileRoute("/console")({
  component: ConsolePage,
});

function ConsolePage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Console</h1>
      <Card padding="sm">
        <LiveConsoleLogViewer />
      </Card>
    </div>
  );
}
