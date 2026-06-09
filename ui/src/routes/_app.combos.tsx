import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  DndContext,
  closestCenter,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/PageHeader";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Icon } from "@/components/common/Icon";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import {
  CardsGridSkeleton,
  ErrorState,
} from "@/components/common/Skeletons";
import { DialogQueryState } from "@/components/common/DialogQueryState";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useState } from "react";
import { toast } from "sonner";
import type { Combo, Model } from "@/lib/types";

export const Route = createFileRoute("/_app/combos")({
  component: CombosPage,
});

function CombosPage() {
  const qc = useQueryClient();
  const {
    data: combos = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery<Combo[]>({
    queryKey: ["combos"],
    queryFn: () => apiFetch("/api/combos"),
  });
  const {
    data: models = [],
    isLoading: modelsLoading,
    isError: modelsError,
    error: modelsErr,
    refetch: refetchModels,
  } = useQuery<Model[]>({
    queryKey: ["models"],
    queryFn: () => apiFetch("/api/models"),
  });

  const [openCreate, setOpenCreate] = useState(false);
  const [editing, setEditing] = useState<Combo | null>(null);
  const [toDelete, setToDelete] = useState<Combo | null>(null);

  const save = useMutation({
    mutationFn: (body: Partial<Combo>) =>
      editing
        ? apiFetch(`/api/combos/${editing.id}`, { method: "PUT", body })
        : apiFetch("/api/combos", { method: "POST", body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["combos"] });
      setOpenCreate(false);
      setEditing(null);
      toast.success("Saved");
    },
  });

  const del = useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/combos/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["combos"] });
      toast.success("Deleted");
    },
  });

  const reorderSteps = (combo: Combo, oldIdx: number, newIdx: number) => {
    const steps = arrayMove(combo.steps ?? [], oldIdx, newIdx);
    save.mutate({ ...combo, steps } as any);
    setEditing({ ...combo, steps });
  };

  return (
    <div>
      <PageHeader
        title="Combos"
        description="Multi-step routing strategies: fallback, round-robin, cheapest, fastest."
        icon="layers"
        actions={
          <Button
            onClick={() => {
              setEditing(null);
              setOpenCreate(true);
            }}
          >
            <Icon name="add" size={16} className="mr-1.5" />
            New combo
          </Button>
        }
      />

      {isLoading ? (
        <CardsGridSkeleton
          count={6}
          height="h-64"
          className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4"
        />
      ) : isError ? (
        <ErrorState
          title="Couldn’t load combos"
          error={error}
          onRetry={() => refetch()}
        />
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {combos.map((c) => (
            <Card key={c.id} className="p-4 card-elev border-border">
              <div className="flex items-start justify-between mb-3">
                <div>
                  <div className="font-semibold">{c.name}</div>
                  <StatusBadge variant="primary" className="mt-1">
                    {c.strategy.replace("_", " ")}
                  </StatusBadge>
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setEditing(c);
                      setOpenCreate(true);
                    }}
                  >
                    <Icon name="edit" size={14} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setToDelete(c)}
                  >
                    <Icon name="delete" size={14} className="text-destructive" />
                  </Button>
                </div>
              </div>

              <SortableSteps
                combo={c}
                onReorder={(oi, ni) => reorderSteps(c, oi, ni)}
              />
            </Card>
          ))}
        </div>
      )}

      <ComboFormDialog
        key={editing?.id ?? "new"}
        open={openCreate}
        editing={editing}
        models={models}
        modelsQuery={{
          isLoading: modelsLoading,
          isError: modelsError,
          error: modelsErr,
          refetch: refetchModels,
        }}
        onSave={(b) => save.mutate(b)}
        onClose={() => {
          setOpenCreate(false);
          setEditing(null);
        }}
      />

      <ConfirmDialog
        open={!!toDelete}
        onOpenChange={(v) => !v && setToDelete(null)}
        title="Delete combo?"
        variant="destructive"
        confirmLabel="Delete"
        onConfirm={() => {
          if (toDelete) del.mutate(toDelete.id);
        }}
      />
    </div>
  );
}

function SortableSteps({
  combo,
  onReorder,
}: {
  combo: Combo;
  onReorder: (o: number, n: number) => void;
}) {
  const steps = combo.steps ?? [];
  const ids = steps.map((_, i) => `${combo.id}-${i}`);
  const onDragEnd = (e: DragEndEvent) => {
    if (e.over && e.active.id !== e.over.id) {
      const oi = ids.indexOf(e.active.id as string);
      const ni = ids.indexOf(e.over.id as string);
      onReorder(oi, ni);
    }
  };
  return (
    <DndContext collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      <SortableContext items={ids} strategy={verticalListSortingStrategy}>
        <div className="space-y-1.5">
          {steps.map((s, i) => (
            <SortableItem key={ids[i]} id={ids[i]} step={s} index={i} />
          ))}
        </div>
      </SortableContext>
    </DndContext>
  );
}

function SortableItem({
  id,
  step,
  index,
}: {
  id: string;
  step: { provider: string; model: string };
  index: number;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id });
  return (
    <div
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.5 : 1,
      }}
      className="flex items-center gap-2 px-2.5 py-2 bg-surface-2 rounded-lg border border-border"
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        className="text-text-muted cursor-grab"
      >
        <Icon name="drag_indicator" size={16} />
      </button>
      <span className="text-xs text-text-muted w-4">{index + 1}.</span>
      <ProviderIcon provider={step.provider} size={20} />
      <span className="font-mono text-xs flex-1 truncate">{step.model}</span>
    </div>
  );
}

function ComboFormDialog({
  open,
  editing,
  models,
  modelsQuery,
  onSave,
  onClose,
}: {
  open: boolean;
  editing: Combo | null;
  models: Model[];
  modelsQuery: {
    isLoading: boolean;
    isError: boolean;
    error?: unknown;
    refetch: () => unknown;
  };
  onSave: (b: any) => void;
  onClose: () => void;
}) {
  const [name, setName] = useState(editing?.name ?? "");
  const [strategy, setStrategy] = useState(editing?.strategy ?? "fallback");
  const [steps, setSteps] = useState(editing?.steps ?? []);

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? "Edit combo" : "New combo"}</DialogTitle>
        </DialogHeader>
        <DialogQueryState
          queries={[modelsQuery]}
          errorTitle="Couldn’t load models"
        >
          <div className="space-y-3">
            <div>
              <label htmlFor="combo-name" className="text-xs font-medium text-text-muted block mb-1">
                Name
              </label>
              <Input id="combo-name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div>
              <label htmlFor="combo-strategy" className="text-xs font-medium text-text-muted block mb-1">
                Strategy
              </label>
              <select
                id="combo-strategy"
                value={strategy}
                onChange={(e) => setStrategy(e.target.value as any)}
                className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm"
              >
                {["fallback", "round_robin", "least_used", "auto", "fastest", "cheapest"].map(
                  (s) => (
                    <option key={s} value={s}>
                      {s.replace("_", " ")}
                    </option>
                  ),
                )}
              </select>
            </div>
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs font-medium text-text-muted">Steps</span>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    const m = models[0];
                    setSteps([...steps, { provider: m.provider, model: m.name }]);
                  }}
                >
                  <Icon name="add" size={14} className="mr-1" />
                  Add step
                </Button>
              </div>
              <div className="space-y-1.5">
                {steps.map((s, i) => (
                  <div key={`step-${i}-${s.provider}-${s.model}`} className="flex items-center gap-1.5">
                    <span className="text-xs w-4 text-text-muted">{i + 1}.</span>
                    <select
                      value={`${s.provider}|${s.model}`}
                      onChange={(e) => {
                        const [provider, model] = e.target.value.split("|");
                        const newSteps = [...steps];
                        newSteps[i] = { provider, model };
                        setSteps(newSteps);
                      }}
                      className="flex-1 bg-surface-2 border border-border rounded-lg px-2 py-1.5 text-xs font-mono"
                    >
                      {models.map((m) => (
                        <option key={m.id} value={`${m.provider}|${m.name}`}>
                          {m.provider} / {m.name}
                        </option>
                      ))}
                    </select>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSteps(steps.filter((_, j) => j !== i))}
                    >
                      <Icon name="close" size={14} />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </DialogQueryState>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={() => onSave({ name, strategy, steps })}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
