import { useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "./Icon";
import { DataTable } from "./DataTable";
import { PageHeader } from "./PageHeader";
import { ConfirmDialog } from "./ConfirmDialog";
import { TableSkeleton, ErrorState } from "./Skeletons";
import { toast } from "sonner";
import type { ColumnDef } from "@tanstack/react-table";

interface Field {
  name: string;
  label: string;
  type?: "text" | "number" | "select" | "textarea" | "switch" | "multiselect" | "date";
  options?: { label: string; value: string }[];
  placeholder?: string;
  required?: boolean;
}

interface Props<T> {
  title: string;
  description?: string;
  icon?: string;
  endpoint: string;
  queryKey: string[];
  columns: ColumnDef<T, any>[];
  fields?: Field[];
  rowKey?: (row: T) => string;
  emptyTitle?: string;
  emptyDescription?: string;
  initialValues?: (row?: T) => Record<string, any>;
  extraToolbar?: ReactNode;
  listSelector?: (resp: any) => T[];
  extraActions?: (row: T) => ReactNode;
  transformBody?: (values: Record<string, any>) => Record<string, any>;
}

export function CrudPage<T extends { id: string | number }>({
  title,
  description,
  icon,
  endpoint,
  queryKey,
  columns,
  fields,
  rowKey,
  emptyTitle,
  emptyDescription,
  initialValues,
  extraToolbar,
  listSelector,
  extraActions,
  transformBody,
}: Props<T>) {
  const qc = useQueryClient();
  const [openForm, setOpenForm] = useState(false);
  const [editing, setEditing] = useState<T | null>(null);
  const [toDelete, setToDelete] = useState<T | null>(null);
  const [values, setValues] = useState<Record<string, any>>({});

  const { data, isLoading, isError, error, refetch } = useQuery<T[]>({
    queryKey,
    queryFn: async () => {
      const r = await apiFetch(endpoint);
      return listSelector ? listSelector(r) : r;
    },
  });

  const save = useMutation({
    mutationFn: async (body: any) => {
      const payload = transformBody ? transformBody(body) : body;
      if (editing) {
        return apiFetch(`${endpoint}/${editing.id}`, { method: "PUT", body: payload });
      }
      return apiFetch(endpoint, { method: "POST", body: payload });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      setOpenForm(false);
      setEditing(null);
      toast.success(editing ? "Updated" : "Created");
    },
    onError: (err: any) => {
      const msg = err?.message || "Failed to save";
      toast.error(msg);
    },
  });

  const del = useMutation({
    mutationFn: (id: string | number) => apiFetch(`${endpoint}/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey });
      toast.success("Deleted");
    },
  });

  const openCreate = () => {
    setEditing(null);
    setValues(initialValues ? initialValues() : {});
    setOpenForm(true);
  };
  const openEdit = (row: T) => {
    setEditing(row);
    setValues(initialValues ? initialValues(row) : (row as any));
    setOpenForm(true);
  };

  const cols: ColumnDef<T, any>[] = [
    ...columns,
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <div className="flex items-center gap-1 justify-end">
          {extraActions?.(row.original)}
          {fields && (
            <Button variant="ghost" size="sm" onClick={() => openEdit(row.original)}>
              <Icon name="edit" size={14} />
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={() => setToDelete(row.original)}>
            <Icon name="delete" size={14} className="text-destructive" />
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title={title}
        description={description}
        icon={icon}
        actions={
          <>
            {extraToolbar}
            {fields && (
              <Button onClick={openCreate} data-testid="create-button">
                <Icon name="add" size={16} className="mr-1.5" />
                Create
              </Button>
            )}
          </>
        }
      />

      {isLoading ? (
        <TableSkeleton rows={6} columns={Math.max(columns.length, 3)} />
      ) : isError ? (
        <ErrorState
          title={`Couldn’t load ${title.toLowerCase()}`}
          error={error}
          onRetry={() => refetch()}
        />
      ) : (
        <DataTable
          columns={cols}
          data={data ?? []}
          emptyTitle={emptyTitle ?? "No records"}
          emptyDescription={emptyDescription}
          emptyIcon={icon}
        />
      )}

      <Dialog open={openForm} onOpenChange={setOpenForm}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit" : "Create"}</DialogTitle>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              save.mutate(values);
            }}
            className="space-y-3"
          >
            {fields?.map((f) => (
              <div key={f.name}>
                <label className="text-xs font-medium block mb-1 text-text-muted">
                  {f.label}
                  {f.required && <span className="text-destructive ml-0.5">*</span>}
                </label>
                {f.type === "textarea" ? (
                  <textarea
                    className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none min-h-[80px]"
                    value={values[f.name] ?? ""}
                    onChange={(e) => setValues((v) => ({ ...v, [f.name]: e.target.value }))}
                    required={f.required}
                  />
                ) : f.type === "multiselect" ? (
                  <div className="flex flex-wrap gap-3">
                    {f.options?.map((o) => {
                      const selected = new Set(values[f.name] ?? []);
                      return (
                        <label
                          key={o.value}
                          className="inline-flex items-center gap-2 text-sm cursor-pointer"
                        >
                          <Checkbox
                            checked={selected.has(o.value)}
                            onCheckedChange={(checked) => {
                              const next = new Set(selected);
                              if (checked) next.add(o.value);
                              else next.delete(o.value);
                              setValues((v) => ({
                                ...v,
                                [f.name]: Array.from(next),
                              }));
                            }}
                          />
                          <span>{o.label}</span>
                        </label>
                      );
                    })}
                  </div>
                ) : f.type === "date" ? (
                  <input
                    type="date"
                    className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                    value={values[f.name] ?? ""}
                    onChange={(e) => setValues((v) => ({ ...v, [f.name]: e.target.value }))}
                    required={f.required}
                  />
                ) : f.type === "select" ? (
                  <select
                    className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                    value={values[f.name] ?? ""}
                    onChange={(e) => setValues((v) => ({ ...v, [f.name]: e.target.value }))}
                    required={f.required}
                  >
                    <option value="">—</option>
                    {f.options?.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                ) : f.type === "switch" ? (
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      className="accent-brand-500 w-4 h-4"
                      checked={!!values[f.name]}
                      onChange={(e) => setValues((v) => ({ ...v, [f.name]: e.target.checked }))}
                    />
                    <span className="text-sm">{f.placeholder ?? f.label}</span>
                  </label>
                ) : (
                  <input
                    type={f.type ?? "text"}
                    className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
                    value={values[f.name] ?? ""}
                    onChange={(e) =>
                      setValues((v) => ({
                        ...v,
                        [f.name]: f.type === "number" ? +e.target.value : e.target.value,
                      }))
                    }
                    placeholder={f.placeholder}
                    required={f.required}
                  />
                )}
              </div>
            ))}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpenForm(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={save.isPending}>
                {save.isPending ? "Saving…" : "Save"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!toDelete}
        onOpenChange={(v) => !v && setToDelete(null)}
        title="Delete record?"
        description="This cannot be undone."
        variant="destructive"
        confirmLabel="Delete"
        onConfirm={() => {
          if (toDelete) del.mutate(toDelete.id);
        }}
      />
    </div>
  );
}
