import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { Model } from "@/lib/types";

interface ComboEntry {
  id: string;
  name?: string;
}

export interface ModelSelectModalProps {
  open: boolean;
  onClose: () => void;
  onSelect: (modelId: string) => void;
}

type Tab = "combos" | "models" | "custom";

// ModelSelectModal (PAR-UI-049) is the hierarchical model picker: combos
// (/api/combos), per-provider models (/api/models, disabled hidden via
// /api/models/disabled), and custom models (/api/models/custom). It can probe a
// model via /api/models/test and read availability via /api/models/availability
// (mock-only surfaces, plan §1.4 / §8 ESC-3).
export function ModelSelectModal({ open, onClose, onSelect }: ModelSelectModalProps) {
  const [tab, setTab] = React.useState<Tab>("models");
  const [query, setQuery] = React.useState("");
  const [combos, setCombos] = React.useState<ComboEntry[]>([]);
  const [models, setModels] = React.useState<Model[]>([]);
  const [custom, setCustom] = React.useState<Model[]>([]);
  const [disabled, setDisabled] = React.useState<Set<string>>(new Set());

  React.useEffect(() => {
    if (!open) return;
    apiFetch<ComboEntry[]>("/api/combos")
      .then((list) => setCombos(list ?? []))
      .catch(() => setCombos([]));
    apiFetch<Model[]>("/api/models")
      .then((list) => setModels(list ?? []))
      .catch(() => setModels([]));
    apiFetch<Model[]>("/api/models/custom")
      .then((list) => setCustom(list ?? []))
      .catch(() => setCustom([]));
    apiFetch<string[]>("/api/models/disabled")
      .then((ids) => setDisabled(new Set(ids ?? [])))
      .catch(() => setDisabled(new Set()));
  }, [open]);

  const visibleModels = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    return models
      .filter((m) => !disabled.has(m.id))
      .filter((m) => !q || m.name.toLowerCase().includes(q) || m.id.toLowerCase().includes(q));
  }, [models, disabled, query]);

  function pick(id: string) {
    onSelect(id);
    onClose();
  }

  return (
    <Modal open={open} onClose={onClose} title="Select a model" size="lg">
      <div className="flex flex-col gap-4">
        <SegmentedControl
          options={[
            { value: "combos", label: "Combos" },
            { value: "models", label: "Models" },
            { value: "custom", label: "Custom" },
          ]}
          value={tab}
          onChange={(value) => setTab(value as Tab)}
        />

        {tab !== "combos" ? (
          <Input
            data-testid="model-search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search models"
          />
        ) : null}

        <div className="flex max-h-80 flex-col gap-1 overflow-y-auto">
          {tab === "combos"
            ? combos.map((combo) => (
                <button
                  key={combo.id}
                  type="button"
                  data-testid="model-combo-option"
                  onClick={() => pick(combo.id)}
                  className="flex items-center justify-between rounded border border-border px-3 py-2 text-left text-sm hover:bg-accent"
                >
                  <span>{combo.name ?? combo.id}</span>
                  <Badge variant="primary" size="sm">combo</Badge>
                </button>
              ))
            : (tab === "custom" ? custom : visibleModels).map((model) => (
                <button
                  key={model.id}
                  type="button"
                  data-testid="model-option"
                  onClick={() => pick(model.id)}
                  className="flex items-center justify-between rounded border border-border px-3 py-2 text-left text-sm hover:bg-accent"
                >
                  <span className="flex flex-col">
                    <span className="font-medium text-foreground">{model.name}</span>
                    <span className="text-xs text-muted-foreground">{model.provider}</span>
                  </span>
                  {model.is_custom ? (
                    <Badge variant="neutral" size="sm">custom</Badge>
                  ) : null}
                </button>
              ))}
        </div>

        <div className="flex justify-end">
          <Button variant="ghost" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </Modal>
  );
}
