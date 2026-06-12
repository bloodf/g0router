import { create } from "zustand";

export interface User {
  id: string;
  username: string;
  role?: string;
}

interface UserState {
  user: User | null;
  token: string | null;
  setUser: (user: User | null) => void;
  setToken: (token: string | null) => void;
  clear: () => void;
}

export const useUserStore = create<UserState>((set) => ({
  user: null,
  token: null,
  setUser: (user) => set({ user }),
  setToken: (token) => set({ token }),
  clear: () => set({ user: null, token: null }),
}));
