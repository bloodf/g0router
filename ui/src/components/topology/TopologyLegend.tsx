export function TopologyLegend() {
  return (
    <div className="absolute bottom-3 left-3 z-10 bg-surface/90 backdrop-blur-md border border-border rounded-lg p-3 shadow-elev text-xs space-y-2 max-w-[220px]">
      <div className="font-semibold text-foreground">Legend</div>
      <div className="space-y-1.5">
        <LegendRow color="bg-info/15 border-info/40" label="API Key" iconColor="text-info" icon="key" />
        <LegendRow color="bg-brand-500/15 border-brand-500/40" label="Combo" iconColor="text-brand-600" icon="layers" />
        <LegendRow color="bg-surface-2 border-border" label="Provider" iconColor="text-foreground" icon="cloud" />
      </div>
      <div className="border-t border-border pt-2 space-y-1.5">
        <div className="flex items-center gap-2">
          <svg width="32" height="6">
            <line
              x1="0"
              y1="3"
              x2="32"
              y2="3"
              stroke="var(--brand-500)"
              strokeWidth="2"
              strokeDasharray="6 4"
            />
          </svg>
          <span>Active (last 30s)</span>
        </div>
        <div className="flex items-center gap-2">
          <svg width="32" height="6">
            <line x1="0" y1="3" x2="32" y2="3" stroke="var(--border)" strokeWidth="1.5" />
          </svg>
          <span>Idle</span>
        </div>
      </div>
      <div className="border-t border-border pt-2 space-y-1">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-success" /> active
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-warning" /> needs re-auth
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-destructive" /> error
        </div>
      </div>
    </div>
  );
}

function LegendRow({
  color,
  label,
  icon,
  iconColor,
}: {
  color: string;
  label: string;
  icon: string;
  iconColor: string;
}) {
  return (
    <div className="flex items-center gap-2">
      <div className={`w-6 h-6 rounded-md border flex items-center justify-center ${color}`}>
        <span className={`material-symbols-outlined ${iconColor}`} style={{ fontSize: 14 }}>
          {icon}
        </span>
      </div>
      <span>{label}</span>
    </div>
  );
}
