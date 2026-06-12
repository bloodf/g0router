import { create } from "zustand";

export interface Toast {
  id: string;
  message: string;
  duration?: number;
}

interface NotificationState {
  toasts: Toast[];
  push: (toast: Omit<Toast, "id"> & { id?: string }) => void;
  dismiss: (id: string) => void;
  clear: () => void;
}

function generateId() {
  return Math.random().toString(36).slice(2);
}

export const useNotificationStore = create<NotificationState>((set) => ({
  toasts: [],
  push: (toast) => {
    const id = toast.id ?? generateId();
    const duration = toast.duration ?? 4000;
    set((state) => ({
      toasts: [...state.toasts, { ...toast, id, duration }],
    }));
    if (duration > 0) {
      setTimeout(() => {
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        }));
      }, duration);
    }
  },
  dismiss: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),
  clear: () => set({ toasts: [] }),
}));
