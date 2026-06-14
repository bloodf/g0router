import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { CardSkeleton } from "@/components/ui/skeleton";
import { PricingModal, type PricingRates } from "@/components/usage/pricing-modal";
import { useNotificationStore } from "@/stores/notification";

export const Route = createFileRoute("/pricing")({
  component: PricingPage,
});

// The real Go GET /api/pricing returns the nested shape
// provider -> model -> {input,output,cached,reasoning,cache_creation}
// (internal/admin/pricing.go:79-95).
type PricingMap = Record<string, Record<string, Partial<PricingRates>>>;

interface PricingRow {
  provider: string;
  model: string;
  rates: PricingRates;
}

function flatten(map: PricingMap): PricingRow[] {
  const rows: PricingRow[] = [];
  for (const provider of Object.keys(map ?? {})) {
    for (const model of Object.keys(map[provider] ?? {})) {
      const r = map[provider][model] ?? {};
      rows.push({
        provider,
        model,
        rates: {
          input: r.input ?? 0,
          output: r.output ?? 0,
          cached: r.cached ?? 0,
          reasoning: r.reasoning ?? 0,
          cache_creation: r.cache_creation ?? 0,
        },
      });
    }
  }
  return rows.sort((a, b) => `${a.provider}/${a.model}`.localeCompare(`${b.provider}/${b.model}`));
}

function PricingPage() {
  const pushToast = useNotificationStore((s) => s.push);
  const [rows, setRows] = React.useState<PricingRow[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<PricingRow | null>(null);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<PricingMap>("/api/pricing")
      .then((data) => {
        setRows(flatten(data));
        setLoading(false);
      })
      .catch(() => {
        setRows([]);
        setLoading(false);
        pushToast({ message: "Failed to load pricing" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Pricing</h1>

      {loading ? (
        <CardSkeleton />
      ) : (
        <Card padding="none">
          <table className="w-full text-sm" data-testid="pricing-table">
            <thead>
              <tr className="border-b border-border text-left text-muted-foreground">
                <th className="px-4 py-2 font-medium">Provider</th>
                <th className="px-4 py-2 font-medium">Model</th>
                <th className="px-4 py-2 font-medium">Input</th>
                <th className="px-4 py-2 font-medium">Output</th>
                <th className="px-4 py-2 font-medium">Cached</th>
                <th className="px-4 py-2 font-medium" />
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={`${row.provider}/${row.model}`} data-testid="pricing-row" className="border-b border-border/50">
                  <td className="px-4 py-2 font-medium text-foreground">{row.provider}</td>
                  <td className="px-4 py-2">{row.model}</td>
                  <td className="px-4 py-2">${row.rates.input.toFixed(2)}</td>
                  <td className="px-4 py-2">${row.rates.output.toFixed(2)}</td>
                  <td className="px-4 py-2">${row.rates.cached.toFixed(2)}</td>
                  <td className="px-4 py-2 text-right">
                    <Button
                      variant="outline"
                      size="sm"
                      data-testid="pricing-edit"
                      onClick={() => setEditing(row)}
                    >
                      Edit
                    </Button>
                  </td>
                </tr>
              ))}
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-6 text-center text-muted-foreground">
                    No pricing overrides configured.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </Card>
      )}

      {editing ? (
        <PricingModal
          open
          onClose={() => setEditing(null)}
          provider={editing.provider}
          model={editing.model}
          rates={editing.rates}
          onSaved={load}
        />
      ) : null}
    </div>
  );
}
