import * as React from "react";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  restrictToVerticalAxis,
  restrictToParentElement,
} from "@dnd-kit/modifiers";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Toggle } from "@/components/ui/toggle";
import { apiFetch } from "@/lib/api";
import { moveStep } from "@/lib/combo-order";
import { useNotificationStore } from "@/stores/notification";
import type { Combo } from "@/lib/types";

export interface ComboFormModalProps {
  open: boolean;
  combo: Combo | null;
  onClose: () => void;
  onSaved?: () => void;
}

type ComboStep = Combo["steps"][number];

const STRATEGY_OPTIONS = [
  { value: "fallback", label: "Fallback" },
  { value: "round_robin", label: "Round robin" },
  { value: "race", label: "Race" },
];

// SortableStep renders one draggable member row. Each row carries the
// combo-step-row test marker and a drag handle (the @dnd-kit listeners).
function SortableStep({ step, id }: { step: ComboStep; id: string }) {
  const { attributes, listeners, setNodeRef, transform, transition } =
    useSortable({ id });
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
  };
  return (
    <div
      ref={setNodeRef}
      style={style}
      data-testid="combo-step-row"
      className="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-2"
    >
      <button
        type="button"
        data-testid="combo-step-handle"
        aria-label={`Reorder ${step.model}`}
        className="cursor-grab text-muted-foreground"
        {...attributes}
        {...listeners}
      >
        <span className="material-symbols-outlined text-base">drag_indicator</span>
      </button>
      <span className="text-sm font-medium text-foreground">{step.model}</span>
      <span className="text-xs text-muted-foreground">{step.provider}</span>
    </div>
  );
}

// ComboFormModal (PAR-UI-050) creates/edits a combo. The member list (steps) is
// reorderable via @dnd-kit; onDragEnd delegates to the pure moveStep helper
// (§1.3). Save persists the members in their current order via
// POST /api/combos (new) or PUT /api/combos/{id} (edit).
function ComboFormModal({ open, combo, onClose, onSaved }: ComboFormModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [strategy, setStrategy] = React.useState("fallback");
  const [isActive, setIsActive] = React.useState(true);
  const [steps, setSteps] = React.useState<ComboStep[]>([]);
  const [busy, setBusy] = React.useState(false);

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  React.useEffect(() => {
    if (combo) {
      setName(combo.name);
      setStrategy(combo.strategy);
      setIsActive(combo.is_active);
      setSteps(combo.steps ?? []);
    } else {
      setName("");
      setStrategy("fallback");
      setIsActive(true);
      setSteps([]);
    }
  }, [combo]);

  function stepId(step: ComboStep, index: number) {
    return `${step.provider}:${step.model}:${index}`;
  }

  function onDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const ids = steps.map(stepId);
    const from = ids.indexOf(String(active.id));
    const to = ids.indexOf(String(over.id));
    setSteps((prev) => moveStep(prev, from, to));
  }

  async function save() {
    setBusy(true);
    const payload = { name, strategy, is_active: isActive, steps };
    try {
      if (combo) {
        await apiFetch(`/api/combos/${combo.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/combos", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: combo ? "Combo updated" : "Combo created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the combo" });
    } finally {
      setBusy(false);
    }
  }

  const ids = steps.map(stepId);

  return (
    <Modal open={open} onClose={onClose} title={combo ? "Edit combo" : "New combo"}>
      <div className="flex flex-col gap-4">
        <Input
          id="combo-name"
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <Select
          id="combo-strategy"
          label="Strategy"
          options={STRATEGY_OPTIONS}
          value={strategy}
          onChange={(event) => setStrategy(event.target.value)}
        />
        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-foreground">Members</span>
          {steps.length === 0 ? (
            <p className="text-xs text-muted-foreground">No members yet.</p>
          ) : (
            <DndContext
              sensors={sensors}
              collisionDetection={closestCenter}
              modifiers={[restrictToVerticalAxis, restrictToParentElement]}
              onDragEnd={onDragEnd}
            >
              <SortableContext items={ids} strategy={verticalListSortingStrategy}>
                <div className="flex flex-col gap-2">
                  {steps.map((step, index) => (
                    <SortableStep key={ids[index]} id={ids[index]} step={step} />
                  ))}
                </div>
              </SortableContext>
            </DndContext>
          )}
        </div>
        <label className="flex items-center justify-between text-sm text-foreground">
          Active
          <Toggle checked={isActive} onCheckedChange={setIsActive} />
        </label>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="combo-save"
            variant="primary"
            loading={busy}
            onClick={save}
          >
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { ComboFormModal };
