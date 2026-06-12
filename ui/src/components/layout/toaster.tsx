import { useEffect, useRef } from "react";
import { Toaster, toast } from "sonner";
import { useNotificationStore } from "@/stores/notification";

export function AppToaster() {
  const toasts = useNotificationStore((state) => state.toasts);
  const forwardedRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    const currentIds = new Set(toasts.map((t) => t.id));

    for (const item of toasts) {
      if (!forwardedRef.current.has(item.id)) {
        toast(item.message, { id: item.id, duration: item.duration });
        forwardedRef.current.add(item.id);
      }
    }

    for (const id of Array.from(forwardedRef.current)) {
      if (!currentIds.has(id)) {
        toast.dismiss(id);
        forwardedRef.current.delete(id);
      }
    }
  }, [toasts]);

  return <Toaster position="bottom-right" />;
}
