import { create } from "zustand";

interface HeaderSearchState {
  query: string;
  setQuery: (query: string) => void;
  clear: () => void;
}

export const useHeaderSearchStore = create<HeaderSearchState>((set) => ({
  query: "",
  setQuery: (query) => set({ query }),
  clear: () => set({ query: "" }),
}));
