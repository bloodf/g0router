import type { ReactNode } from "react";
import { Icon } from "./Icon";

export function PageHeader({
  title,
  description,
  icon,
  actions,
}: {
  title: string;
  description?: string;
  icon?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-4 mb-6 flex-wrap">
      <div className="min-w-0">
        <h1 className="text-2xl font-semibold tracking-tight flex items-center gap-2">
          {icon && <Icon name={icon} size={26} className="text-brand-500" />}
          {title}
        </h1>
        {description && (
          <p className="text-sm text-text-muted mt-1 max-w-3xl">{description}</p>
        )}
      </div>
      {actions && <div className="flex items-center gap-2 flex-wrap">{actions}</div>}
    </div>
  );
}
