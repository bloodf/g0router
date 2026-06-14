import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { UsageStats, type UsagePeriod } from "@/components/usage/usage-stats";
import { RequestLogger } from "@/components/usage/request-logger";
import { RequestDetailsTab } from "@/components/usage/request-details-tab";
import { UsageCharts } from "@/components/usage/usage-charts";

export const Route = createFileRoute("/usage")({
  component: UsagePage,
});

type Tab = "overview" | "logs" | "details";

const TAB_OPTIONS = [
  { value: "overview", label: "Overview" },
  { value: "logs", label: "Logs" },
  { value: "details", label: "Details" },
];

function UsagePage() {
  const [tab, setTab] = React.useState<Tab>("overview");
  const [period, setPeriod] = React.useState<UsagePeriod>("7d");

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Usage</h1>

      <div data-testid="usage-tabs">
        <SegmentedControl options={TAB_OPTIONS} value={tab} onChange={(v) => setTab(v as Tab)} />
      </div>

      {tab === "overview" ? (
        <div className="flex flex-col gap-6">
          <UsageStats period={period} setPeriod={setPeriod} />
          <UsageCharts period={period} />
        </div>
      ) : null}

      {tab === "logs" ? <RequestLogger /> : null}

      {tab === "details" ? <RequestDetailsTab /> : null}
    </div>
  );
}
