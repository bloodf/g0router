import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Icon } from "../common/Icon";

export type AuthTypeFilter = "all" | "oauth" | "api_key" | "noauth";
export type StatusFilter = "all" | "active" | "needs_reauth" | "error" | "inactive";
export type TimeWindow = 30 | 120 | 300;

export interface TopologyFilters {
  auth_type: AuthTypeFilter;
  status: StatusFilter;
  window_sec: TimeWindow;
}

export const DEFAULT_FILTERS: TopologyFilters = {
  auth_type: "all",
  status: "all",
  window_sec: 30,
};

interface Props {
  value: TopologyFilters;
  onChange: (v: TopologyFilters) => void;
}

export function TopologyFilterBar({ value, onChange }: Props) {
  const reset = () => onChange(DEFAULT_FILTERS);
  const dirty =
    value.auth_type !== "all" ||
    value.status !== "all" ||
    value.window_sec !== 30;

  return (
    <div className="absolute top-3 right-3 z-10 flex items-center gap-1 bg-surface/90 backdrop-blur-md border border-border rounded-lg p-1 shadow-elev">
      <div className="flex items-center gap-1 px-1.5 text-xs text-text-muted">
        <Icon name="filter_alt" size={14} />
      </div>
      <Select
        value={value.auth_type}
        onValueChange={(v) => onChange({ ...value, auth_type: v as AuthTypeFilter })}
      >
        <SelectTrigger className="h-7 w-[120px] text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All auth</SelectItem>
          <SelectItem value="oauth">OAuth</SelectItem>
          <SelectItem value="api_key">API key</SelectItem>
          <SelectItem value="noauth">No auth</SelectItem>
        </SelectContent>
      </Select>
      <Select
        value={value.status}
        onValueChange={(v) => onChange({ ...value, status: v as StatusFilter })}
      >
        <SelectTrigger className="h-7 w-[130px] text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All status</SelectItem>
          <SelectItem value="active">Active</SelectItem>
          <SelectItem value="needs_reauth">Needs re-auth</SelectItem>
          <SelectItem value="error">Error</SelectItem>
          <SelectItem value="inactive">Inactive</SelectItem>
        </SelectContent>
      </Select>
      <Select
        value={String(value.window_sec)}
        onValueChange={(v) =>
          onChange({ ...value, window_sec: Number(v) as TimeWindow })
        }
      >
        <SelectTrigger className="h-7 w-[90px] text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="30">30s</SelectItem>
          <SelectItem value="120">2m</SelectItem>
          <SelectItem value="300">5m</SelectItem>
        </SelectContent>
      </Select>
      {dirty && (
        <Button variant="ghost" size="sm" className="gap-1 h-7" onClick={reset}>
          <Icon name="close" size={14} />
          Reset
        </Button>
      )}
    </div>
  );
}
