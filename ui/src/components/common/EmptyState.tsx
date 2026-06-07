import { Icon } from "./Icon";
import { Button } from "@/components/ui/button";
import type { ReactNode } from "react";

export function EmptyState({
  icon = "inbox",
  title,
  description,
  action,
}: {
  icon?: string;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void; icon?: string };
  children?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
      <div className="w-16 h-16 rounded-2xl bg-surface-2 flex items-center justify-center mb-4">
        <Icon name={icon} size={32} className="text-text-muted" />
      </div>
      <h3 className="text-base font-semibold">{title}</h3>
      {description && (
        <p className="mt-1.5 text-sm text-text-muted max-w-md">{description}</p>
      )}
      {action && (
        <Button onClick={action.onClick} className="mt-5">
          {action.icon && <Icon name={action.icon} size={16} className="mr-1.5" />}
          {action.label}
        </Button>
      )}
    </div>
  );
}
