import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Icon } from "../common/Icon";
import { ProviderIcon } from "../common/ProviderIcon";
import { StatusBadge } from "../common/StatusBadge";

export function KeyNode({ data, selected }: NodeProps) {
  return (
    <div
      className={
        "bg-surface rounded-xl border px-3 py-2 shadow-elev w-[200px] " +
        (selected ? "border-brand-500 shadow-warm" : "border-border")
      }
    >
      <div className="flex items-center gap-2">
        <div className="w-8 h-8 rounded-lg bg-info/10 text-info flex items-center justify-center">
          <Icon name="key" size={16} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-sm font-semibold truncate">{(data as any).label}</div>
          <div className="text-[10px] font-mono text-text-muted truncate">
            {(data as any).prefix}
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Right} className="!bg-brand-500" />
    </div>
  );
}

export function ComboNode({ data, selected }: NodeProps) {
  return (
    <div
      className={
        "bg-surface rounded-xl border px-3 py-2 shadow-elev w-[200px] " +
        (selected ? "border-brand-500 shadow-warm" : "border-border")
      }
    >
      <div className="flex items-center gap-2">
        <div className="w-8 h-8 rounded-lg bg-brand-500/10 text-brand-600 flex items-center justify-center">
          <Icon name="layers" size={16} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-sm font-semibold truncate">{(data as any).label}</div>
          <div className="text-[10px] text-text-muted">{(data as any).strategy}</div>
        </div>
      </div>
      <Handle type="target" position={Position.Left} className="!bg-brand-500" />
      <Handle type="source" position={Position.Right} className="!bg-brand-500" />
    </div>
  );
}

export function ProviderNode({ data, selected }: NodeProps) {
  const status = (data as any).status as string;
  const variant =
    status === "active"
      ? "success"
      : status === "error"
        ? "danger"
        : status === "needs_reauth"
          ? "warning"
          : "muted";
  return (
    <div
      className={
        "bg-surface rounded-xl border px-3 py-2 shadow-elev w-[200px] " +
        (selected ? "border-brand-500 shadow-warm" : "border-border")
      }
    >
      <div className="flex items-center gap-2">
        <ProviderIcon provider={(data as any).provider} size={32} />
        <div className="min-w-0 flex-1">
          <div className="text-sm font-semibold truncate">{(data as any).label}</div>
          <StatusBadge variant={variant as any} dot className="mt-0.5 text-[9px]">
            {(data as any).connectionCount} conn
          </StatusBadge>
        </div>
      </div>
      <Handle type="target" position={Position.Left} className="!bg-brand-500" />
    </div>
  );
}

export const nodeTypes = {
  key: KeyNode,
  combo: ComboNode,
  provider: ProviderNode,
};
