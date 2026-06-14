import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import type { Combo } from "@/lib/types";

export interface ComboListProps {
  combos: Combo[];
  onToggle: (combo: Combo, active: boolean) => void;
  onEdit: (combo: Combo) => void;
  onDelete: (combo: Combo) => void;
}

// ComboList (PAR-PR-339) renders the combos list view: one row per combo with
// the name, strategy badge, member count, an active Toggle, and Edit/Delete.
function ComboList({ combos, onToggle, onEdit, onDelete }: ComboListProps) {
  return (
    <div className="flex flex-col gap-2">
      {combos.map((combo) => (
        <div
          key={combo.id}
          data-testid="combo-row"
          className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
        >
          <div className="flex items-center gap-3">
            <div>
              <p className="text-sm font-medium text-foreground">{combo.name}</p>
              <p className="text-xs text-muted-foreground">
                {combo.steps.length} member{combo.steps.length === 1 ? "" : "s"}
              </p>
            </div>
            <Badge variant="default" size="sm">
              {combo.strategy}
            </Badge>
          </div>
          <div className="flex items-center gap-2">
            <Toggle
              checked={combo.is_active}
              onCheckedChange={(checked) => onToggle(combo, checked)}
              aria-label={`Toggle ${combo.name}`}
            />
            <Button
              data-testid="combo-edit"
              variant="ghost"
              size="sm"
              onClick={() => onEdit(combo)}
            >
              Edit
            </Button>
            <Button
              data-testid="combo-delete"
              variant="danger"
              size="sm"
              onClick={() => onDelete(combo)}
            >
              Delete
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}

export { ComboList };
