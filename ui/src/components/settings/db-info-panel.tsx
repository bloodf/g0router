import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface DbInfo {
  path: string;
  size_bytes: number;
  tables: { name: string; rows: number }[];
}

function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

// DbInfoPanel (PAR-UI-101) displays the database path/size/table counts from the
// mock GET /api/settings/database (no real Go endpoint yet, plan §1.4/§8 ESC-3).
export function DbInfoPanel() {
  const [info, setInfo] = React.useState<DbInfo | null>(null);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    let active = true;
    apiFetch<DbInfo>("/api/settings/database")
      .then((data) => {
        if (active) setInfo(data ?? null);
      })
      .catch(() => {
        if (active) setInfo(null);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Database</CardTitle>
      </CardHeader>
      <CardContent className="mt-4">
        <div data-testid="db-info-panel" className="flex flex-col gap-3">
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading database info…</p>
          ) : info ? (
            <>
              <div className="text-sm text-foreground">
                <span className="font-medium">Path: </span>
                <span className="font-mono text-xs">{info.path}</span>
              </div>
              <div className="text-sm text-foreground">
                <span className="font-medium">Size: </span>
                <span>{formatBytes(info.size_bytes)}</span>
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-sm font-medium text-foreground">Tables</span>
                {info.tables.map((table) => (
                  <div
                    key={table.name}
                    className="flex justify-between text-xs text-muted-foreground"
                  >
                    <span className="font-mono">{table.name}</span>
                    <span>{table.rows} rows</span>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">Database info unavailable.</p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
