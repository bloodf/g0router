import { create } from "zustand";

export interface Provider {
  id: string;
  name: string;
  kind: string;
  enabled: boolean;
}

interface ProviderState {
  providers: Provider[];
  setProviders: (providers: Provider[]) => void;
  upsertProvider: (provider: Provider) => void;
  removeProvider: (id: string) => void;
}

export const useProviderStore = create<ProviderState>((set) => ({
  providers: [],
  setProviders: (providers) => set({ providers }),
  upsertProvider: (provider) =>
    set((state) => {
      const index = state.providers.findIndex((p) => p.id === provider.id);
      if (index === -1) {
        return { providers: [...state.providers, provider] };
      }
      const next = [...state.providers];
      next[index] = provider;
      return { providers: next };
    }),
  removeProvider: (id) =>
    set((state) => ({
      providers: state.providers.filter((p) => p.id !== id),
    })),
}));
