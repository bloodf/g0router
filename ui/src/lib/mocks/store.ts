import { create } from "zustand";
import { seedAll, type Store } from "./seed";

interface MockState {
  store: Store;
  reset: () => void;
  simulateFirstRun: () => void;
  setSession: (userId: string | null) => void;
  patchStore: <K extends keyof Store>(key: K, value: Store[K]) => void;
}

export const useMockStore = create<MockState>((set) => ({
  store: seedAll(),
  reset: () => set({ store: seedAll() }),
  simulateFirstRun: () =>
    set((s) => ({ store: { ...s.store, users: [], session_user_id: null as any } })),
  setSession: (uid) =>
    set((s) => ({ store: { ...s.store, session_user_id: uid as any } })),
  patchStore: (key, value) =>
    set((s) => ({ store: { ...s.store, [key]: value } })),
}));

export const getStore = () => useMockStore.getState().store;
export const setStore = (mutator: (s: Store) => Store) =>
  useMockStore.setState((state) => ({ store: mutator(state.store) }));
